package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jelinden/newsfeedreader/app/domain"
	"github.com/jelinden/newsfeedreader/app/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type M map[string]interface{}
type S []M

const (
	dbName       = "news"
	collection   = "newscollection"
	fetchTimeout = 5 * time.Second
)

type Mongo struct {
	Client *mongo.Client

	mostReadCache       map[string][]domain.RSS
	mostReadCacheExpiry map[string]time.Time
	mostReadCacheTTL    time.Duration
	mostReadCacheMutex  sync.RWMutex
}

// NewMongo connects to MongoDB and verifies the connection with a ping.
// A non-nil error means the application cannot serve any data.
func NewMongo(mongoAddress string) (*Mongo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoAddress))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	m := &Mongo{
		Client:              client,
		mostReadCache:       make(map[string][]domain.RSS),
		mostReadCacheExpiry: make(map[string]time.Time),
		mostReadCacheTTL:    1 * time.Minute,
	}
	go m.createIndexes()
	return m, nil
}

func (m *Mongo) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Client.Disconnect(ctx); err != nil {
		log.Println("mongo disconnect error:", err)
	}
}

func (m *Mongo) collection() *mongo.Collection {
	return m.Client.Database(dbName).Collection(collection)
}

// createIndexes creates the query indexes, retrying for a bounded period so a
// slow-to-start database does not leave the deployment without indexes.
func (m *Mongo) createIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "language", Value: 1}, {Key: "pubDate", Value: -1}}},
		{Keys: bson.D{{Key: "language", Value: 1}, {Key: "category.categoryName", Value: 1}}},
		{Keys: bson.D{{Key: "language", Value: 1}, {Key: "rssSource", Value: 1}}},
		{Keys: bson.D{{Key: "language", Value: 1}, {Key: "pubDate", Value: -1}, {Key: "clicks", Value: -1}}},
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		_, err := m.collection().Indexes().CreateMany(ctx, indexes)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			log.Println("failed to create indexes:", err)
			return
		case <-ticker.C:
		}
	}

	// The text index backs $text searches. It is created separately because
	// MongoDB allows only one text index per collection.
	m.ensureTextIndex(ctx)
}

// textLanguageOverride points MongoDB at a field that does not exist in our
// documents. Without it MongoDB interprets the document's "language" field
// ("fi"/"en") as an analyzer name; those are not valid MongoDB language names,
// which makes text indexing unreliable on recent servers.
const textLanguageOverride = "mongoTextLanguage"

const textIndexName = "news_text"

// ensureTextIndex creates the text index only when the collection has none.
// An existing text index is never dropped or replaced: production has
// rssTitle_text_rssDesc_text serving live traffic, and rebuilding a text index
// on a large collection is expensive and temporarily breaks search. Set
// NEWS_REPLACE_TEXT_INDEX=true to migrate deliberately.
func (m *Mongo) ensureTextIndex(ctx context.Context) {
	desired := mongo.IndexModel{
		Keys: bson.D{
			{Key: "rssTitle", Value: "text"},
			{Key: "rssDesc", Value: "text"},
		},
		Options: options.Index().
			SetName(textIndexName).
			SetDefaultLanguage("english").
			SetLanguageOverride(textLanguageOverride),
	}

	existing, err := m.collection().Indexes().List(ctx)
	if err != nil {
		log.Println("failed to list indexes:", err)
		return
	}
	defer existing.Close(ctx)

	var specs []bson.M
	if err := existing.All(ctx, &specs); err != nil {
		log.Println("failed to decode index specs:", err)
		return
	}

	replace := os.Getenv("NEWS_REPLACE_TEXT_INDEX") == "true"
	for _, spec := range specs {
		key, _ := spec["key"].(bson.M)
		if _, isText := key["_fts"]; !isText {
			continue
		}
		name, _ := spec["name"].(string)
		if name == textIndexName && !replace {
			return // our index is already in place
		}
		if !replace {
			log.Printf("keeping existing text index %q; set NEWS_REPLACE_TEXT_INDEX=true to rebuild it", name)
			return
		}
		if _, err := m.collection().Indexes().DropOne(ctx, name); err != nil {
			log.Println("failed to drop text index", name, ":", err)
			return
		}
		log.Println("dropped text index", name, "(NEWS_REPLACE_TEXT_INDEX=true)")
		break
	}

	if _, err := m.collection().Indexes().CreateOne(ctx, desired); err != nil {
		log.Println("failed to create text index:", err)
		return
	}
	log.Println("indexes created successfully")
}

func (m *Mongo) FetchRssItems(lang string, page int, count int) []domain.RSS {
	query := M{
		"language":              lang,
		"category.categoryName": M{"$nin": []string{"Mobiili", "Blogs"}},
	}
	result := m.query(query, page, count)
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) FetchRssItemsByCategory(lang string, category string, page int, count int) []domain.RSS {
	query := M{
		"language":              lang,
		"category.categoryName": category,
	}
	result := m.query(query, page, count)
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) FetchRssItemsBySource(lang string, source string, page int, count int) []domain.RSS {
	query := M{
		"language":  lang,
		"rssSource": source,
	}
	result := m.query(query, page, count)
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

// MostReadWeekly returns the most clicked news of the past week. The result
// set is cached per language (the cache key has no page component, so this
// always returns the top `count` items); every caller receives its own deep
// copy so concurrent requests never share - and mutate - the same items.
func (m *Mongo) MostReadWeekly(lang string, count int) []domain.RSS {
	result, cached := m.cacheGet(lang)
	if !cached {
		result = m.fetchMostReadWeekly(lang, count)
		m.cacheSet(lang, result)
	}

	out := copyRSS(result)
	if lang == "en" {
		out = util.AddCategoryEnNames(out)
	}
	return out
}

func (m *Mongo) fetchMostReadWeekly(lang string, count int) []domain.RSS {
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -7)
	query := M{
		"language": lang,
		"pubDate":  M{"$gt": dateFrom, "$lt": dateTo},
	}
	limit := int64(count)
	findOptions := options.Find().
		SetLimit(limit).
		SetSort(bson.D{{Key: "clicks", Value: -1}, {Key: "pubDate", Value: -1}})

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	result := []domain.RSS{}
	cursor, err := m.collection().Find(ctx, query, findOptions)
	if err != nil {
		log.Println("mostReadWeekly query failed:", err)
		return result
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &result); err != nil {
		log.Println("mostReadWeekly decode failed:", err)
		return []domain.RSS{}
	}
	return result
}

func (m *Mongo) cacheGet(key string) ([]domain.RSS, bool) {
	m.mostReadCacheMutex.RLock()
	defer m.mostReadCacheMutex.RUnlock()

	result, exists := m.mostReadCache[key]
	if !exists || time.Now().After(m.mostReadCacheExpiry[key]) {
		return nil, false
	}
	return result, true
}

func (m *Mongo) cacheSet(key string, result []domain.RSS) {
	m.mostReadCacheMutex.Lock()
	defer m.mostReadCacheMutex.Unlock()

	m.mostReadCache[key] = result
	m.mostReadCacheExpiry[key] = time.Now().Add(m.mostReadCacheTTL)
}

// copyRSS deep-copies the slice and its items so callers can mutate the
// returned entries without touching the cache.
func copyRSS(items []domain.RSS) []domain.RSS {
	out := make([]domain.RSS, len(items))
	copy(out, items)
	return out
}

// Search runs a mongo $text query. Note that $language must NOT be set here:
// MongoDB applies the *index* default language (see ensureTextIndex) as the
// query analyzer as well, and passing e.g. "finnish" would stem the search
// terms differently from the indexed text and silently return zero hits.
// Exact term matching therefore works for both "fi" and "en" documents.
func (m *Mongo) Search(searchString string, lang string, page int, count int) []domain.RSS {
	query := M{
		"$text":    M{"$search": searchString},
		"language": lang,
	}
	result := m.query(query, page, count)
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) query(query map[string]interface{}, page int, count int) []domain.RSS {
	limit := int64(count)
	skip := int64(page * count)
	findOptions := options.Find().
		SetLimit(limit).
		SetSkip(skip).
		SetSort(bson.D{{Key: "pubDate", Value: -1}})

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	result := []domain.RSS{}
	cursor, err := m.collection().Find(ctx, query, findOptions)
	if err != nil {
		log.Println("query failed:", err)
		return result
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &result); err != nil {
		log.Println("query decode failed:", err)
		return []domain.RSS{}
	}
	return result
}

// IsValidID reports whether id can be used as a mongo ObjectID.
func IsValidID(id string) bool {
	_, err := primitive.ObjectIDFromHex(id)
	return err == nil
}

// SaveClick increments the click counter of a news item.
func (m *Mongo) SaveClick(id string) error {
	itemId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid object id %q: %w", id, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	_, err = m.collection().UpdateOne(ctx,
		bson.D{{Key: "_id", Value: itemId}},
		M{"$inc": M{"clicks": 1}})
	if err != nil {
		return fmt.Errorf("click update failed for %q: %w", id, err)
	}
	return nil
}

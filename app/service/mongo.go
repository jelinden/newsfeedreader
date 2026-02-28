package service

import (
	"context"
	"log"
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

type Mongo struct {
	Client                *mongo.Client
	mostReadCache         map[string][]domain.RSS
	mostReadCacheMutex    sync.RWMutex
	mostReadCacheExpiry   map[string]time.Time
	mostReadCacheTTL      time.Duration
	indexesCreated        bool
	indexesMutex          sync.Mutex
}

var mongoConn Mongo

func NewMongo(mongoAddress string) *Mongo {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoAddress))
	if err != nil {
		log.Println("connection lost ", err)
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Println("mongo connection failed ", err)
	}
	mongoConn = Mongo{
		Client:              client,
		mostReadCache:       make(map[string][]domain.RSS),
		mostReadCacheExpiry: make(map[string]time.Time),
		mostReadCacheTTL:    1 * time.Hour,
		indexesCreated:      false,
	}
	go mongoConn.createIndexes()
	return &mongoConn
}

func (m *Mongo) Close() {
	m.Client.Disconnect(context.Background())
}

func (m *Mongo) createIndexes() {
	m.indexesMutex.Lock()
	defer m.indexesMutex.Unlock()
	if m.indexesCreated {
		return
	}
	c := m.Client.Database("news").Collection("newscollection")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	indexModel1 := mongo.IndexModel{
		Keys: bson.D{{Key: "language", Value: 1}, {Key: "pubDate", Value: -1}},
	}
	indexModel2 := mongo.IndexModel{
		Keys: bson.D{{Key: "language", Value: 1}, {Key: "category.categoryName", Value: 1}},
	}
	indexModel3 := mongo.IndexModel{
		Keys: bson.D{{Key: "language", Value: 1}, {Key: "rssSource", Value: 1}},
	}
	indexModel4 := mongo.IndexModel{
		Keys: bson.D{{Key: "language", Value: 1}, {Key: "pubDate", Value: -1}, {Key: "clicks", Value: -1}},
	}
	
	_, err := c.Indexes().CreateMany(ctx, []mongo.IndexModel{indexModel1, indexModel2, indexModel3, indexModel4})
	if err != nil {
		log.Println("failed to create indexes:", err)
	} else {
		m.indexesCreated = true
		log.Println("indexes created successfully")
	}
}

func (m *Mongo) FetchRssItems(lang string, from int, count int) []domain.RSS {
	query := M{
		"language":              lang,
		"category.categoryName": M{"$nin": []string{"Mobiili", "Blogs"}},
	}
	result := m.query(query, from, count)
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) FetchRssItemsByCategory(lang string, category string, from int, count int) []domain.RSS {
	query := M{
		"language":              lang,
		"category.categoryName": category,
	}
	result := m.query(query, from, count)
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) FetchRssItemsBySource(lang string, source string, from int, count int) []domain.RSS {
	query := M{
		"language":  lang,
		"rssSource": source,
	}
	result := m.query(query, from, count)
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) MostReadWeekly(lang string, from int, count int) []domain.RSS {
	cacheKey := lang
	
	m.mostReadCacheMutex.RLock()
	if cachedResult, exists := m.mostReadCache[cacheKey]; exists {
		if time.Now().Before(m.mostReadCacheExpiry[cacheKey]) {
			m.mostReadCacheMutex.RUnlock()
			return cachedResult
		}
	}
	m.mostReadCacheMutex.RUnlock()
	
	result := []domain.RSS{}
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -7)
	c := mongoConn.Client.Database("news").Collection("newscollection")
	query := M{
		"language": lang,
		"pubDate":  M{"$gt": dateFrom, "$lt": dateTo},
	}
	limit := int64(count)
	skip := int64(from * count)
	findOptions := options.FindOptions{
		Limit: &limit,
		Sort:  bson.D{{Key: "clicks", Value: -1}, {Key: "pubDate", Value: -1}},
		Skip:  &skip,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	cursor, _ := c.Find(ctx, query, &findOptions)
	if err := cursor.All(ctx, &result); err != nil {
		log.Println(err)
	}
	defer cursor.Close(ctx)
	
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	
	m.mostReadCacheMutex.Lock()
	m.mostReadCache[cacheKey] = result
	m.mostReadCacheExpiry[cacheKey] = time.Now().Add(m.mostReadCacheTTL)
	m.mostReadCacheMutex.Unlock()
	
	return result
}

func (m *Mongo) Search(searchString string, lang string, from int, count int) []domain.RSS {
	query := M{
		"$text":    M{"$search": "\"" + searchString + "\"", "$language": lang},
		"language": lang,
	}

	result := []domain.RSS{}
	c := mongoConn.Client.Database("news").Collection("newscollection")

	limit := int64(count)
	skip := int64(from * count)
	findOptions := options.FindOptions{
		Limit: &limit,
		Sort:  bson.D{{Key: "pubDate", Value: -1}},
		Skip:  &skip,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	cursor, err := c.Find(ctx, query, &findOptions)
	if err != nil {
		log.Println("search failed", err)
		return result
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &result); err != nil {
		log.Println(err)
	}

	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) query(query map[string]interface{}, from int, count int) []domain.RSS {
	result := []domain.RSS{}
	c := mongoConn.Client.Database("news").Collection("newscollection")

	limit := int64(count)
	skip := int64(from * count)
	findOptions := options.FindOptions{
		Limit: &limit,
		Sort:  bson.D{{Key: "pubDate", Value: -1}},
		Skip:  &skip,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	cursor, _ := c.Find(ctx, query, &findOptions)
	if err := cursor.All(ctx, &result); err != nil {
		log.Println(err)
	}
	defer cursor.Close(ctx)
	return result
}

func (m *Mongo) SaveClick(id string) {
	c := mongoConn.Client.Database("news").Collection("newscollection")
	itemId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Println("saving click error", id, err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err = c.UpdateOne(ctx, bson.D{{Key: "_id", Value: itemId}}, M{"$inc": M{"clicks": 1}})
	if err != nil {
		log.Println("upsert error", id, err.Error())
	}
}

func News(searchString string) []domain.RSS {
	var result = []domain.RSS{}
	query := M{
		"$text":    M{"$search": `"` + searchString + `"`, "$language": "en"},
		"language": "en",
	}
	limit := int64(20)
	findOptions := options.FindOptions{
		Limit: &limit,
		Sort:  bson.D{{Key: "pubDate", Value: -1}},
	}
	c := mongoConn.Client.Database("news").Collection("newscollection")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	cursor, _ := c.Find(ctx, query, &findOptions)
	if err := cursor.All(ctx, &result); err != nil {
		log.Println(err)
	}
	defer cursor.Close(ctx)
	return result
}

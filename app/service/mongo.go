package service

import (
	"log"
	"os"
	"time"

	"github.com/jelinden/newsfeedreader/app/domain"
	"github.com/jelinden/newsfeedreader/app/util"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type M map[string]interface{}
type S []M
type Mongo struct {
	mongo *mgo.Session
}

func NewMongo() *Mongo {
	m := &Mongo{}
	m.mongo = m.createSession(os.Getenv("MONGO_URL"))
	return m
}

func (m *Mongo) Close() {
	m.mongo.Close()
}

func (m *Mongo) createSession(url string) *mgo.Session {
	maxWait := time.Duration(5 * time.Second)
	session, err := mgo.DialWithTimeout(url, maxWait)
	if err != nil {
		log.Println("mongo connection lost")
	}
	session.SetSocketTimeout(30 * time.Second)
	session.SetMode(mgo.Monotonic, true)
	return session
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
	result := []domain.RSS{}
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -7)
	sess := m.mongo.Clone()
	defer sess.Close()
	c := sess.DB("news").C("newscollection")
	query := M{
		"language": lang,
		"pubDate":  M{"$gt": dateFrom, "$lt": dateTo},
	}
	err := c.Find(query).Select(M{"rssDesc": 0}).Sort("-clicks", "-pubDate").Skip(from * count).Limit(count).All(&result)
	if err != nil {
		log.Println("Mongo error " + err.Error())
	}
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) Search(searchString string, lang string, from int, count int) []domain.RSS {
	query := M{
		"$text":    M{"$search": "\"" + searchString + "\"", "$language": lang},
		"language": lang,
	}

	result := []domain.RSS{}
	sess := m.mongo.Clone()
	defer sess.Close()
	c := sess.DB("news").C("newscollection")
	err := c.Find(query).Select(M{"rssDesc": 0}).Select(bson.M{"score": bson.M{"$meta": "textScore"}}).Sort("-pubDate", "$textScore:score").Skip(from * count).Limit(count).All(&result)
	if err != nil {
		log.Println("Mongo error " + err.Error())
	}
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) query(query map[string]interface{}, from int, count int) []domain.RSS {
	result := []domain.RSS{}
	sess := m.mongo.Clone()
	defer sess.Close()
	c := sess.DB("news").C("newscollection")
	err := c.Find(query).Select(M{"rssDesc": 0}).Sort("-pubDate").Skip(from * count).Limit(count).All(&result)
	if err != nil {
		log.Println("Mongo error " + err.Error())
	}
	return result
}

func (m *Mongo) SaveClick(id string) {
	s := m.mongo.Clone()
	defer s.Close()
	c := s.DB("news").C("newscollection")
	_, err := c.UpsertId(bson.ObjectIdHex(id), M{"$inc": M{"clicks": 1}})
	if err != nil {
		log.Println("upsert error", id, err.Error())
	}
}

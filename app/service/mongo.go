package service

import (
	"fmt"
	"github.com/jelinden/newsfeedreader/app/domain"
	"github.com/jelinden/newsfeedreader/app/util"
	mgo "gopkg.in/mgo.v2"
	"os"
	"time"
)

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
		fmt.Println("connection lost")
	}
	session.SetSocketTimeout(30 * time.Second)
	session.SetMode(mgo.Monotonic, true)
	return session
}

func (m *Mongo) FetchRssItems(lang string, from int, count int) []domain.RSS {
	type M map[string]interface{}
	query := M{
		"language":              lang,
		"category.categoryName": M{"$ne": "Mobiili"},
	}
	result := m.query(query, from, count)
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) Search(searchString string, lang string, from int, count int) []domain.RSS {
	type M map[string]interface{}
	query := M{
		"$text": M{"$search": searchString, "$language": lang},
		//"category.categoryName": M{"$ne": "Mobiili"},
	}
	result := m.query(query, from, count)
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

func (m *Mongo) query(query map[string]interface{}, from int, count int) []domain.RSS {
	result := []domain.RSS{}
	type M map[string]interface{}
	sess := m.mongo.Clone()
	c := sess.DB("news").C("newscollection")
	err := c.Find(query).Select(M{"rssDesc": 0}).Sort("-pubDate").Skip(from * count).Limit(count).All(&result)
	if err != nil {
		fmt.Println("Mongo error " + err.Error())
	}
	sess.Close()
	return result
}

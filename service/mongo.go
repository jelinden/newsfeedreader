package service

import (
	"fmt"
	"github.com/jelinden/newsfeedreader/domain"
	"github.com/jelinden/newsfeedreader/util"
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
	session.SetMode(mgo.Monotonic, true)
	return session
}

func (m *Mongo) FetchRssItems(lang string, from int, count int) []domain.RSS {
	result := []domain.RSS{}
	type M map[string]interface{}
	sess := m.mongo.Clone()
	c := sess.DB("news").C("newscollection")
	err := c.Find(M{
		"language":              lang,
		"category.categoryName": M{"$ne": "Mobiili"},
	}).Sort("-pubDate").Skip(from).Limit(count).All(&result)
	if err != nil {
		fmt.Println("Mongo error " + err.Error())
	}
	if lang == "en" {
		result = util.AddCategoryEnNames(result)
	}
	return result
}

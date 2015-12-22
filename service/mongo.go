package service

import (
	"fmt"
	"github.com/jelinden/newsfeedreader/domain"
	"github.com/jelinden/newsfeedreader/util"
	mgo "gopkg.in/mgo.v2"
	"os"
	"time"
)

type Sessions struct {
	Mongo *mgo.Session
}

func NewSessions() *Sessions {
	return &Sessions{}
}

func (s *Sessions) Init() {
	s.Mongo = s.createSession(os.Getenv("MONGO_URL"))
}

func (s *Sessions) Close() {
	s.Mongo.Close()
}

func (s *Sessions) createSession(url string) *mgo.Session {
	maxWait := time.Duration(5 * time.Second)
	session, err := mgo.DialWithTimeout(url, maxWait)
	if err != nil {
		fmt.Println("connection lost")
	}
	session.SetMode(mgo.Monotonic, true)
	return session
}

func (s *Sessions) FetchRssItems(lang string, from int, count int) []domain.RSS {
	result := []domain.RSS{}
	type M map[string]interface{}
	sess := s.Mongo.Clone()
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

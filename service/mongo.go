package service

import (
	"fmt"
	"os"
	"time"

	"github.com/jelinden/newsfeedreader/domain"
	mgo "gopkg.in/mgo.v2"
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

func (s *Sessions) FetchRssItems(lang string) []domain.RSS {
	result := []domain.RSS{}
	type M map[string]interface{}
	sess := s.Mongo.Clone()
	c := sess.DB("news").C("newscollection")
	err := c.Find(M{
		"language":              lang,
		"category.categoryName": M{"$ne": "Mobiili"},
	}).Sort("-pubDate").Limit(30).All(&result)
	if err != nil {
		fmt.Println("Mongo error " + err.Error())
	}
	return result
}

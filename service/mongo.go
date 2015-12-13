package service

import (
	"fmt"
	"os"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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

func (s *Sessions) FetchRssItems(lang int) map[string]interface{} {
	result := []domain.RSS{}
	sess := s.Mongo.Clone()
	c := sess.DB("uutispuro").C("rss")
	err := c.Find(bson.M{"language": lang}).Sort("-date").Limit(30).All(&result)
	if err != nil {
		fmt.Println("Fatal error " + err.Error())
	}
	return map[string]interface{}{"news": result}
}

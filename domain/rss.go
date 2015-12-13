package domain

import (
	"gopkg.in/mgo.v2/bson"
)

type RSS struct {
	Id        bson.ObjectId `json:"id" bson:"_id"`
	RssTitle  string
	RssLink   string
	PubDate   time.Time
	RssSource string
	RssClicks int64
	Language  String
	Category  Category
	RssFeed   RssFeed
}

type Category struct {
	Id           bson.ObjectId `json:"id" bson:"_id"`
	CategoryName string
}

type RssFeed struct {
	Id        bson.ObjectId `json:"id" bson:"_id"`
	Url       string
	SiteUrl   String
	FeedTitle String
}

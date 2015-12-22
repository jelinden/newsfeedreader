package domain

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type RSS struct {
	Id        bson.ObjectId `json:"id" bson:"_id"`
	RssTitle  string        `json:"rssTitle" bson:"rssTitle"`
	RssLink   string        `json:"rssLink" bson:"rssLink"`
	PubDate   time.Time     `json:"pubDate" bson:"pubDate"`
	RssSource string        `json:"rssSource" bson:"rssSource"`
	RssClicks int64         `json:"rssClicks" bson:"rssClicks"`
	Language  string        `json:"language" bson:"language"`
	Category  Category      `json:"category" bson:"category"`
	RssFeed   RssFeed       `json:"rssFeed" bson:"rssFeed"`
}

type Category struct {
	Id             bson.ObjectId `json:"id" bson:"_id"`
	CategoryName   string        `json:"categoryName" bson:"categoryName"`
	CategoryEnName string        `json:"categoryEnName" bson:"-"`
}

type RssFeed struct {
	Id        bson.ObjectId `json:"id" bson:"_id"`
	Url       string        `json:"url" bson:"url"`
	SiteUrl   string        `json:"siteUrl" bson:"siteUrl"`
	FeedTitle string        `json:"feedTitle" bson:"feedTitle"`
}

package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RSS struct {
	Id        primitive.ObjectID `json:"id" bson:"_id"`
	RssTitle  string             `json:"rssTitle" bson:"rssTitle"`
	RssLink   string             `json:"rssLink" bson:"rssLink"`
	PubDate   time.Time          `json:"pubDate" bson:"pubDate"`
	RssSource string             `json:"rssSource" bson:"rssSource"`
	Clicks    int                `json:"-" bson:"clicks"`
	Language  string             `json:"language" bson:"language"`
	Category  Category           `json:"category" bson:"category"`
	RssFeed   RssFeed            `json:"-" bson:"rssFeed"`
}

type Category struct {
	Id             primitive.ObjectID `json:"id" bson:"_id"`
	CategoryName   string             `json:"categoryName" bson:"categoryName"`
	CategoryEnName string             `json:"categoryEnName" bson:"enName"`
}

type RssFeed struct {
	Id        primitive.ObjectID `json:"id" bson:"_id"`
	Url       string             `json:"url" bson:"url"`
	SiteUrl   string             `json:"siteUrl" bson:"siteUrl"`
	FeedTitle string             `json:"feedTitle" bson:"feedTitle"`
}

type News struct {
	RSS            []RSS  `json:"rssList"`
	MostReadList   []RSS  `json:"mostReadList"`
	Page           int    `json:"page"`
	Lang           string `json:"lang"`
	SearchQuery    string `json:"searchQuery,omitempty"`
	ResultCount    int    `json:"count"`
	Category       string `json:"category,omitempty"`
	CategoryEnName string `json:"categoryEnName,omitempty" bson:"-"`
	Source         string `json:"source,omitempty" bson:"-"`
}

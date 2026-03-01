package tick

import (
	"encoding/json"
	"log"
	"time"

	socketio "github.com/googollee/go-socket.io"
	"github.com/jelinden/newsfeedreader/app/service"
)

type Tick struct {
	Mongo          *service.Mongo
	NewsFi, NewsEn string
}

func NewTick(mongo *service.Mongo) *Tick {
	tick := &Tick{}
	tick.Mongo = mongo
	return tick
}

func (t *Tick) TickNews(lang string) {
	var lastNews string
	for range time.Tick(10 * time.Second) {
		rssList := t.Mongo.FetchRssItems(lang, 0, 5)
		if len(rssList) > 0 {
			result := map[string]interface{}{"news": rssList}
			news, err := json.Marshal(result)
			if err != nil {
				log.Println(err.Error())
			} else {
				newsStr := string(news)
				if newsStr != lastNews {
					lastNews = newsStr
					if lang == "fi" {
						t.NewsFi = newsStr
					} else {
						t.NewsEn = newsStr
					}
				}
			}
		} else {
			log.Println("Fetched rss", lang, "list was empty")
		}
	}
}

func (t *Tick) TickEmit(server *socketio.Server) {
	for _ = range time.Tick(10 * time.Second) {
		server.BroadcastToNamespace("en", "message", t.NewsEn)
		server.BroadcastToNamespace("fi", "message", t.NewsFi)
	}
}

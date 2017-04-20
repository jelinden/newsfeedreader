package tick

import (
	"encoding/json"
	"log"
	"time"

	"github.com/googollee/go-socket.io"
	"github.com/jelinden/newsfeedreader/app/service"
)

type Tick struct {
	Mongo          *service.Mongo
	newsFi, newsEn string
}

func NewTick(mongo *service.Mongo) *Tick {
	tick := &Tick{}
	tick.Mongo = mongo
	return tick
}

func (t *Tick) TickNews(lang string) {
	for _ = range time.Tick(10 * time.Second) {
		rssList := t.Mongo.FetchRssItems(lang, 0, 5)
		if len(rssList) > 0 {
			result := map[string]interface{}{"news": rssList}
			news, err := json.Marshal(result)
			if err != nil {
				log.Println(err.Error())
			} else {
				if lang == "fi" {
					t.newsFi = string(news)
				} else {
					t.newsEn = string(news)
				}
			}
		} else {
			log.Println("Fetched rss list was empty")
		}
	}
}

func (t *Tick) TickEmit(server *socketio.Server) {
	for _ = range time.Tick(10 * time.Second) {
		server.BroadcastTo("en", "message", t.newsEn)
		server.BroadcastTo("fi", "message", t.newsFi)
	}
}

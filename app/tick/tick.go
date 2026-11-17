package tick

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/jelinden/newsfeedreader/app/service"
)

const (
	interval = 10 * time.Second
	feedSize = 5
	// emptyPayload keeps the websocket contract valid before the first poll
	// completes, so clients never have to parse an empty frame.
	emptyPayload = `{"news":[]}`
)

type Tick struct {
	Mongo *service.Mongo

	newsMutex sync.RWMutex
	newsFi    string
	newsEn    string
}

func NewTick(mongo *service.Mongo) *Tick {
	return &Tick{Mongo: mongo, newsFi: emptyPayload, newsEn: emptyPayload}
}

// TickNews polls the newest news for one language and publishes them to the
// websocket broadcast state whenever the payload changes.
func (t *Tick) TickNews(lang string) {
	var lastNews string
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		rssList := t.Mongo.FetchRssItems(lang, 0, feedSize)
		if len(rssList) == 0 {
			log.Println("Fetched rss", lang, "list was empty")
			continue
		}
		news, err := json.Marshal(map[string]interface{}{"news": rssList})
		if err != nil {
			log.Println("marshal error:", err.Error())
			continue
		}
		newsStr := string(news)
		if newsStr == lastNews {
			continue
		}
		lastNews = newsStr
		t.SetNews(lang, newsStr)
	}
}

// SetNews stores the latest broadcast payload for a language ("fi" or "en").
func (t *Tick) SetNews(lang, news string) {
	t.newsMutex.Lock()
	defer t.newsMutex.Unlock()
	if lang == "fi" {
		t.newsFi = news
	} else {
		t.newsEn = news
	}
}

// Latest returns the current broadcast payload for a language ("fi" or "en").
// The result is always valid JSON.
func (t *Tick) Latest(lang string) string {
	t.newsMutex.RLock()
	defer t.newsMutex.RUnlock()
	if lang == "fi" {
		return t.newsFi
	}
	return t.newsEn
}

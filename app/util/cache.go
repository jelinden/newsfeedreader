package util

import (
	"time"

	"github.com/streamrail/concurrent-map"
)

var Cache cmap.ConcurrentMap

const cacheSize = 10000

type CacheItem struct {
	Key    string
	Value  []byte
	Type   string
	Expire time.Time
}

func init() {
	Cache = cmap.New()
}

// AddItemToCache add a new item to cache with expiration time
func AddItemToCache(key string, itemType string, value []byte, expire time.Duration) {
	if Cache.Count() <= cacheSize {
		Cache.Set(key, CacheItem{Key: key, Type: itemType, Value: value, Expire: time.Now().Add(expire)})
	} else {
		panic("cacheSize reached")
	}
}

// GetItemFromCache get item from cache
func GetItemFromCache(key string) *CacheItem {
	if Cache.Has(key) {
		if tmp, ok := Cache.Get(key); ok {
			item := tmp.(CacheItem)
			if item.Expire.After(time.Now()) {
				return &item
			}
			removeItem(item.Key)
		}
	}
	return nil
}

func removeItem(key string) {
	Cache.Remove(key)
}

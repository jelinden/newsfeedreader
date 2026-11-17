package service

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jelinden/newsfeedreader/app/domain"
)

func newTestMongo(ttl time.Duration) *Mongo {
	return &Mongo{
		mostReadCache:       make(map[string][]domain.RSS),
		mostReadCacheExpiry: make(map[string]time.Time),
		mostReadCacheTTL:    ttl,
	}
}

func TestCacheGetMissAndHit(t *testing.T) {
	m := newTestMongo(1 * time.Hour)

	if _, ok := m.cacheGet("fi"); ok {
		t.Error("empty cache must report a miss")
	}

	data := []domain.RSS{{RssTitle: "Test", Language: "fi"}}
	m.cacheSet("fi", data)

	got, ok := m.cacheGet("fi")
	if !ok {
		t.Fatal("cacheGet should hit after cacheSet")
	}
	if len(got) != 1 || got[0].RssTitle != "Test" {
		t.Errorf("unexpected cached data: %+v", got)
	}
}

func TestCacheExpiry(t *testing.T) {
	m := newTestMongo(50 * time.Millisecond)
	m.cacheSet("fi", []domain.RSS{{RssTitle: "Test"}})

	if _, ok := m.cacheGet("fi"); !ok {
		t.Error("cache should be fresh immediately after set")
	}
	time.Sleep(80 * time.Millisecond)
	if _, ok := m.cacheGet("fi"); ok {
		t.Error("cache entry should have expired")
	}
}

// TestCacheGetReturnsIsolatedData verifies the bug where callers received the
// shared cached slice and AddCategoryEnNames mutated it concurrently.
func TestCacheGetReturnsIsolatedData(t *testing.T) {
	m := newTestMongo(1 * time.Hour)
	m.cacheSet("en", []domain.RSS{{RssTitle: "Test", Category: domain.Category{CategoryName: "Digi"}}})

	cached, _ := m.cacheGet("en")
	first := copyRSS(cached)
	second := copyRSS(cached)

	// Simulate what the request path does: mutate one copy...
	first[0].Category.CategoryEnName = "Digital media"
	if second[0].Category.CategoryEnName != "" {
		t.Error("mutating one returned copy affected another (shared backing array)")
	}
	if cached[0].Category.CategoryEnName != "" {
		t.Error("mutating a copy affected the cache")
	}
}

func TestConcurrentCacheAccess(t *testing.T) {
	// Run with `go test -race` to detect cache data races.
	m := newTestMongo(1 * time.Hour)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			m.cacheSet("fi", []domain.RSS{{RssTitle: "x"}})
		}()
		go func() {
			defer wg.Done()
			if got, ok := m.cacheGet("fi"); ok {
				_ = copyRSS(got)
			}
		}()
	}
	wg.Wait()
}

func TestIsValidID(t *testing.T) {
	valid := []string{"507f1f77bcf86cd799439011"}
	invalid := []string{"", "zz7f1f77bcf86cd799439011", "507f1f77bcf86cd79943901", "not-an-id"}

	for _, id := range valid {
		if !IsValidID(id) {
			t.Errorf("IsValidID(%q) = false, want true", id)
		}
	}
	for _, id := range invalid {
		if IsValidID(id) {
			t.Errorf("IsValidID(%q) = true, want false", id)
		}
	}
}

// TestSaveClickRejectsInvalidID ensures malformed ids never reach the
// database (previously an invalid id produced an update with a zero ObjectID).
func TestSaveClickRejectsInvalidID(t *testing.T) {
	m := &Mongo{} // no client: any DB access would panic
	err := m.SaveClick("garbage")
	if err == nil {
		t.Fatal("expected error for invalid id")
	}
	if !strings.Contains(err.Error(), "invalid object id") {
		t.Errorf("unexpected error: %v", err)
	}
}

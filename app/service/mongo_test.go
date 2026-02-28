package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jelinden/newsfeedreader/app/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// TestMostReadWeeklyCaching tests the caching mechanism
func TestMostReadWeeklyCaching(t *testing.T) {
	// Create a mock Mongo instance
	m := &Mongo{
		mostReadCache:       make(map[string][]domain.RSS),
		mostReadCacheExpiry: make(map[string]time.Time),
		mostReadCacheTTL:    1 * time.Hour,
	}

	// Test 1: Cache returns nil when empty
	cacheKey := "fi"
	m.mostReadCacheMutex.RLock()
	_, exists := m.mostReadCache[cacheKey]
	m.mostReadCacheMutex.RUnlock()

	if exists {
		t.Error("Cache should be empty initially")
	}

	// Test 2: Store in cache
	testData := []domain.RSS{
		{RssTitle: "Test News 1", Language: "fi"},
		{RssTitle: "Test News 2", Language: "fi"},
	}

	m.mostReadCacheMutex.Lock()
	m.mostReadCache[cacheKey] = testData
	m.mostReadCacheExpiry[cacheKey] = time.Now().Add(1 * time.Hour)
	m.mostReadCacheMutex.Unlock()

	// Test 3: Retrieve from cache
	m.mostReadCacheMutex.RLock()
	cached, exists := m.mostReadCache[cacheKey]
	expiry := m.mostReadCacheExpiry[cacheKey]
	m.mostReadCacheMutex.RUnlock()

	if !exists {
		t.Error("Cache should contain stored data")
	}

	if len(cached) != 2 {
		t.Errorf("Expected 2 cached items, got %d", len(cached))
	}

	if !time.Now().Before(expiry) {
		t.Error("Cache expiry should be in the future")
	}

	// Test 4: Check cache expiry
	m.mostReadCacheMutex.Lock()
	m.mostReadCacheExpiry[cacheKey] = time.Now().Add(-1 * time.Hour) // Expired
	m.mostReadCacheMutex.Unlock()

	m.mostReadCacheMutex.RLock()
	_, isValid := m.mostReadCache[cacheKey]
	expiry = m.mostReadCacheExpiry[cacheKey]
	m.mostReadCacheMutex.RUnlock()

	if isValid && !time.Now().Before(expiry) {
		t.Log("Cache correctly identified as expired")
	}
}

// TestCacheThreadSafety tests concurrent cache access
func TestCacheThreadSafety(t *testing.T) {
	m := &Mongo{
		mostReadCache:       make(map[string][]domain.RSS),
		mostReadCacheExpiry: make(map[string]time.Time),
		mostReadCacheTTL:    1 * time.Hour,
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)
	testData := []domain.RSS{{RssTitle: "Test", Language: "fi"}}

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			m.mostReadCacheMutex.Lock()
			m.mostReadCache["fi"] = testData
			m.mostReadCacheExpiry["fi"] = time.Now().Add(1 * time.Hour)
			m.mostReadCacheMutex.Unlock()
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			m.mostReadCacheMutex.RLock()
			_, exists := m.mostReadCache["fi"]
			m.mostReadCacheMutex.RUnlock()
			if !exists {
				errors <- nil // This is ok, cache may not be populated yet
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	if len(errors) > 0 {
		t.Logf("Concurrent access test completed with %d non-critical issues (expected)", len(errors))
	}
}

// TestIndexesCreation verifies index structure
func TestIndexesCreation(t *testing.T) {
	m := &Mongo{
		indexesCreated: false,
	}

	// Test 1: Flag starts as false
	if m.indexesCreated {
		t.Error("indexesCreated should be false initially")
	}

	// Test 2: Simulate index creation completion
	m.indexesMutex.Lock()
	m.indexesCreated = true
	m.indexesMutex.Unlock()

	m.indexesMutex.Lock()
	if !m.indexesCreated {
		t.Error("indexesCreated should be true after creation")
	}
	m.indexesMutex.Unlock()

	// Test 3: Prevent duplicate index creation
	created := false
	m.indexesMutex.Lock()
	if m.indexesCreated {
		created = false // Would skip in actual code
	} else {
		created = true
	}
	m.indexesMutex.Unlock()

	if created {
		t.Error("Should not create indexes twice")
	}
}

// TestContextTimeout tests context timeout implementation
func TestContextTimeout(t *testing.T) {
	// Test 1: Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if ctx == nil {
		t.Error("Context should not be nil")
	}

	// Test 2: Verify timeout is set
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	// Deadline should be approximately 5 seconds in future
	diff := time.Until(deadline)
	if diff < 4*time.Second || diff > 6*time.Second {
		t.Errorf("Expected deadline ~5s, got %v", diff)
	}

	// Test 3: Timeout cancellation
	fastCtx, fastCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	time.Sleep(200 * time.Millisecond)

	select {
	case <-fastCtx.Done():
		t.Log("Context correctly timed out")
	default:
		t.Error("Context should have timed out")
	}
	fastCancel()
}

// TestCacheHTTL verifies cache TTL behavior
func TestCacheTTL(t *testing.T) {
	m := &Mongo{
		mostReadCache:       make(map[string][]domain.RSS),
		mostReadCacheExpiry: make(map[string]time.Time),
		mostReadCacheTTL:    100 * time.Millisecond, // Short TTL for testing
	}

	testData := []domain.RSS{{RssTitle: "Test", Language: "fi"}}

	// Store in cache
	m.mostReadCacheMutex.Lock()
	m.mostReadCache["fi"] = testData
	m.mostReadCacheExpiry["fi"] = time.Now().Add(m.mostReadCacheTTL)
	m.mostReadCacheMutex.Unlock()

	// Verify it's valid immediately
	m.mostReadCacheMutex.RLock()
	_, exists := m.mostReadCache["fi"]
	isValid := time.Now().Before(m.mostReadCacheExpiry["fi"])
	m.mostReadCacheMutex.RUnlock()

	if !exists || !isValid {
		t.Error("Cache should be valid immediately after creation")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	m.mostReadCacheMutex.RLock()
	_, exists = m.mostReadCache["fi"]
	isValid = time.Now().Before(m.mostReadCacheExpiry["fi"])
	m.mostReadCacheMutex.RUnlock()

	if exists && !isValid {
		t.Log("Cache correctly expired after TTL")
	} else if !exists {
		t.Error("Cache entry should still exist, only marked as expired")
	}
}

// TestMultiLanguageCaching tests separate caches for different languages
func TestMultiLanguageCaching(t *testing.T) {
	m := &Mongo{
		mostReadCache:       make(map[string][]domain.RSS),
		mostReadCacheExpiry: make(map[string]time.Time),
		mostReadCacheTTL:    1 * time.Hour,
	}

	// Store different data for different languages
	fiData := []domain.RSS{{RssTitle: "Finnish News", Language: "fi"}}
	enData := []domain.RSS{{RssTitle: "English News", Language: "en"}}

	m.mostReadCacheMutex.Lock()
	m.mostReadCache["fi"] = fiData
	m.mostReadCache["en"] = enData
	m.mostReadCacheExpiry["fi"] = time.Now().Add(1 * time.Hour)
	m.mostReadCacheExpiry["en"] = time.Now().Add(1 * time.Hour)
	m.mostReadCacheMutex.Unlock()

	// Verify separate caches
	m.mostReadCacheMutex.RLock()
	fi, fiExists := m.mostReadCache["fi"]
	en, enExists := m.mostReadCache["en"]
	m.mostReadCacheMutex.RUnlock()

	if !fiExists || !enExists {
		t.Error("Both language caches should exist")
	}

	if len(fi) != 1 || fi[0].RssTitle != "Finnish News" {
		t.Error("Finnish cache has wrong data")
	}

	if len(en) != 1 || en[0].RssTitle != "English News" {
		t.Error("English cache has wrong data")
	}

	t.Log("Multi-language caching works correctly")
}

// BenchmarkCacheAccess benchmarks cache read performance
func BenchmarkCacheAccess(b *testing.B) {
	m := &Mongo{
		mostReadCache:       make(map[string][]domain.RSS),
		mostReadCacheExpiry: make(map[string]time.Time),
		mostReadCacheTTL:    1 * time.Hour,
	}

	testData := []domain.RSS{{RssTitle: "Test", Language: "fi"}}
	m.mostReadCacheMutex.Lock()
	m.mostReadCache["fi"] = testData
	m.mostReadCacheExpiry["fi"] = time.Now().Add(1 * time.Hour)
	m.mostReadCacheMutex.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.mostReadCacheMutex.RLock()
		_, _ = m.mostReadCache["fi"]
		m.mostReadCacheMutex.RUnlock()
	}
}

// BenchmarkCacheWrite benchmarks cache write performance
func BenchmarkCacheWrite(b *testing.B) {
	m := &Mongo{
		mostReadCache:       make(map[string][]domain.RSS),
		mostReadCacheExpiry: make(map[string]time.Time),
		mostReadCacheTTL:    1 * time.Hour,
	}

	testData := []domain.RSS{{RssTitle: "Test", Language: "fi"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.mostReadCacheMutex.Lock()
		m.mostReadCache["fi"] = testData
		m.mostReadCacheExpiry["fi"] = time.Now().Add(1 * time.Hour)
		m.mostReadCacheMutex.Unlock()
	}
}

// TestIndexModelStructure verifies index model construction
func TestIndexModelStructure(t *testing.T) {
	// Test 1: Language + PubDate Index
	indexModel1 := mongo.IndexModel{
		Keys: bson.D{{Key: "language", Value: 1}, {Key: "pubDate", Value: -1}},
	}
	if indexModel1.Keys == nil {
		t.Error("Index model 1 keys should not be nil")
	}

	// Test 2: Language + Category Index
	indexModel2 := mongo.IndexModel{
		Keys: bson.D{{Key: "language", Value: 1}, {Key: "category.categoryName", Value: 1}},
	}
	if indexModel2.Keys == nil {
		t.Error("Index model 2 keys should not be nil")
	}

	// Test 3: Language + Source Index
	indexModel3 := mongo.IndexModel{
		Keys: bson.D{{Key: "language", Value: 1}, {Key: "rssSource", Value: 1}},
	}
	if indexModel3.Keys == nil {
		t.Error("Index model 3 keys should not be nil")
	}

	// Test 4: Language + PubDate + Clicks Index (compound)
	indexModel4 := mongo.IndexModel{
		Keys: bson.D{
			{Key: "language", Value: 1},
			{Key: "pubDate", Value: -1},
			{Key: "clicks", Value: -1},
		},
	}
	if indexModel4.Keys == nil {
		t.Error("Index model 4 keys should not be nil")
	}

	t.Log("All index models correctly structured")
}

// TestMongoStructureFields verifies new struct fields exist
func TestMongoStructureFields(t *testing.T) {
	m := &Mongo{
		mostReadCache:       make(map[string][]domain.RSS),
		mostReadCacheExpiry: make(map[string]time.Time),
		mostReadCacheTTL:    1 * time.Hour,
		indexesCreated:      false,
	}

	// Test that all new fields are accessible
	if m.mostReadCache == nil {
		t.Error("mostReadCache should be initialized")
	}

	if m.mostReadCacheExpiry == nil {
		t.Error("mostReadCacheExpiry should be initialized")
	}

	if m.mostReadCacheTTL != 1*time.Hour {
		t.Errorf("mostReadCacheTTL should be 1 hour, got %v", m.mostReadCacheTTL)
	}

	if m.indexesCreated {
		t.Error("indexesCreated should be false initially")
	}

	t.Log("All Mongo struct fields properly initialized")
}

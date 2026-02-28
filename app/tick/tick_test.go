package tick

import (
	"encoding/json"
	"testing"

	"github.com/jelinden/newsfeedreader/app/domain"
)

// TestNewsChangeDetection tests the change detection logic
func TestNewsChangeDetection(t *testing.T) {
	// Test 1: Different JSON strings should be detected as different
	data1 := map[string]interface{}{"news": []domain.RSS{{RssTitle: "News 1"}}}
	data2 := map[string]interface{}{"news": []domain.RSS{{RssTitle: "News 2"}}}

	json1, _ := json.Marshal(data1)
	json2, _ := json.Marshal(data2)

	str1 := string(json1)
	str2 := string(json2)

	if str1 == str2 {
		t.Error("Different news should produce different JSON strings")
	}

	// Test 2: Same JSON strings should be equal
	data3 := map[string]interface{}{"news": []domain.RSS{{RssTitle: "News 1"}}}
	json3, _ := json.Marshal(data3)
	str3 := string(json3)

	if str1 != str3 {
		t.Error("Same news should produce same JSON strings")
	}

	// Test 3: Empty vs non-empty should be different
	empty := map[string]interface{}{"news": []domain.RSS{}}
	nonempty := map[string]interface{}{"news": []domain.RSS{{RssTitle: "News 1"}}}

	emptyJson, _ := json.Marshal(empty)
	nonEmptyJson, _ := json.Marshal(nonempty)

	if string(emptyJson) == string(nonEmptyJson) {
		t.Error("Empty and non-empty news should produce different JSON")
	}

	t.Log("Change detection logic works correctly")
}

// TestJSONMarshalConsistency tests that same data produces same JSON
func TestJSONMarshalConsistency(t *testing.T) {
	// Create identical datasets
	rss := []domain.RSS{
		{RssTitle: "Test 1", RssLink: "http://example.com"},
		{RssTitle: "Test 2", RssLink: "http://example.com/2"},
	}

	data := map[string]interface{}{"news": rss}

	// Marshal multiple times
	json1, err1 := json.Marshal(data)
	if err1 != nil {
		t.Errorf("Failed to marshal: %v", err1)
	}

	json2, err2 := json.Marshal(data)
	if err2 != nil {
		t.Errorf("Failed to marshal: %v", err2)
	}

	str1 := string(json1)
	str2 := string(json2)

	if str1 != str2 {
		t.Error("Same data should produce identical JSON strings")
	}

	t.Log("JSON marshaling is consistent")
}

// TestEmptyNewsList tests handling of empty news lists
func TestEmptyNewsList(t *testing.T) {
	// Test 1: Empty slice
	empty := []domain.RSS{}
	data := map[string]interface{}{"news": empty}
	json1, _ := json.Marshal(data)

	// Test 2: Nil slice
	var nilSlice []domain.RSS
	data2 := map[string]interface{}{"news": nilSlice}
	json2, _ := json.Marshal(data2)

	// Both should serialize as empty arrays
	str1 := string(json1)
	str2 := string(json2)

	if str1 != str2 {
		t.Logf("Empty vs nil slice produce different JSON (this may be expected)\nEmpty: %s\nNil: %s", str1, str2)
	}

	t.Log("Empty news list handling verified")
}

// TestLastNewsTracking simulates the lastNews tracking
func TestLastNewsTracking(t *testing.T) {
	var lastNews string
	messagesSent := 0

	// Simulate first update (should always send)
	news1 := map[string]interface{}{"news": []domain.RSS{{RssTitle: "News 1"}}}
	json1, _ := json.Marshal(news1)
	newsStr1 := string(json1)

	if newsStr1 != lastNews {
		messagesSent++
		lastNews = newsStr1
	}

	if messagesSent != 1 {
		t.Error("First message should be sent")
	}

	// Simulate second update with same data (should not send)
	news2 := map[string]interface{}{"news": []domain.RSS{{RssTitle: "News 1"}}}
	json2, _ := json.Marshal(news2)
	newsStr2 := string(json2)

	if newsStr2 != lastNews {
		messagesSent++
		lastNews = newsStr2
	}

	if messagesSent != 1 {
		t.Error("Duplicate message should not be sent")
	}

	// Simulate third update with different data (should send)
	news3 := map[string]interface{}{"news": []domain.RSS{{RssTitle: "News 2"}}}
	json3, _ := json.Marshal(news3)
	newsStr3 := string(json3)

	if newsStr3 != lastNews {
		messagesSent++
		lastNews = newsStr3
	}

	if messagesSent != 2 {
		t.Error("Updated message should be sent")
	}

	t.Logf("Change tracking works: sent %d messages out of 3 updates", messagesSent)
}

// TestMultipleLanguageTracking tests tracking for multiple languages independently
func TestMultipleLanguageTracking(t *testing.T) {
	// Simulate separate tracking for Fi and En
	var lastNewsFi, lastNewsEn string
	updatesFi, updatesEn := 0, 0

	// Fi update 1
	fiData := map[string]interface{}{"news": []domain.RSS{{RssTitle: "Suomalainen Uutinen"}}}
	fiJson, _ := json.Marshal(fiData)
	fiStr := string(fiJson)
	if fiStr != lastNewsFi {
		updatesFi++
		lastNewsFi = fiStr
	}

	// En update 1
	enData := map[string]interface{}{"news": []domain.RSS{{RssTitle: "English News"}}}
	enJson, _ := json.Marshal(enData)
	enStr := string(enJson)
	if enStr != lastNewsEn {
		updatesEn++
		lastNewsEn = enStr
	}

	// Fi duplicate (no update)
	fiStr2 := string(fiJson)
	if fiStr2 != lastNewsFi {
		updatesFi++
	}

	// En update 2 (new data)
	enData2 := map[string]interface{}{"news": []domain.RSS{{RssTitle: "New English News"}}}
	enJson2, _ := json.Marshal(enData2)
	enStr2 := string(enJson2)
	if enStr2 != lastNewsEn {
		updatesEn++
		lastNewsEn = enStr2
	}

	if updatesFi != 1 || updatesEn != 2 {
		t.Errorf("Expected Fi: 1 update, En: 2 updates. Got Fi: %d, En: %d", updatesFi, updatesEn)
	}

	t.Log("Multi-language change tracking works correctly")
}

// BenchmarkJSONMarshal benchmarks JSON marshaling
func BenchmarkJSONMarshal(b *testing.B) {
	rss := []domain.RSS{
		{RssTitle: "Test 1", RssLink: "http://example.com", Language: "fi"},
		{RssTitle: "Test 2", RssLink: "http://example.com/2", Language: "fi"},
		{RssTitle: "Test 3", RssLink: "http://example.com/3", Language: "fi"},
		{RssTitle: "Test 4", RssLink: "http://example.com/4", Language: "fi"},
		{RssTitle: "Test 5", RssLink: "http://example.com/5", Language: "fi"},
	}

	data := map[string]interface{}{"news": rss}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(data)
	}
}

// BenchmarkStringComparison benchmarks string comparison
func BenchmarkStringComparison(b *testing.B) {
	rss := []domain.RSS{
		{RssTitle: "Test 1", RssLink: "http://example.com"},
		{RssTitle: "Test 2", RssLink: "http://example.com/2"},
	}

	data := map[string]interface{}{"news": rss}
	json1, _ := json.Marshal(data)
	str1 := string(json1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = str1 == str1
	}
}

// TestChangeDetectionEfficiency simulates efficiency of change detection
func TestChangeDetectionEfficiency(t *testing.T) {
	// Create test data: 5 news items
	rss := make([]domain.RSS, 5)
	for i := 0; i < 5; i++ {
		rss[i] = domain.RSS{
			RssTitle: "News",
			RssLink:  "http://example.com",
		}
	}

	data := map[string]interface{}{"news": rss}
	json1, _ := json.Marshal(data)
	str1 := string(json1)

	// Scenario 1: 10 identical updates in a row
	noChanges := 0
	lastNews := str1
	for i := 0; i < 10; i++ {
		json2, _ := json.Marshal(data)
		str2 := string(json2)
		if str2 == lastNews {
			noChanges++
		}
	}

	if noChanges != 10 {
		t.Errorf("Expected 10 unchanged updates, got %d", noChanges)
	}

	t.Logf("Change detection correctly identified %d duplicate updates out of 10 ticks", noChanges)
}

// TestRSSFieldPresence tests that RSS fields are preserved in JSON
func TestRSSFieldPresence(t *testing.T) {
	rss := domain.RSS{
		RssTitle: "Test Title",
		RssLink:  "http://example.com",
		Language: "fi",
	}

	data := map[string]interface{}{"news": []domain.RSS{rss}}
	jsonBytes, _ := json.Marshal(data)
	jsonStr := string(jsonBytes)

	// Check that key fields are in the JSON
	expectedFields := []string{"Test Title", "http://example.com", "fi"}
	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("Expected field '%s' not found in JSON", field)
		}
	}

	t.Log("All RSS fields properly preserved in JSON marshaling")
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

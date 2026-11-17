package tick

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/jelinden/newsfeedreader/app/domain"
)

// TestSetNewsAndLatest exercises the real Tick accessors used by the
// websocket handlers and the polling goroutine.
func TestSetNewsAndLatest(t *testing.T) {
	tr := NewTick(nil)

	// A fresh tick must hand out valid JSON so clients never parse "".
	if got := tr.Latest("fi"); got != `{"news":[]}` {
		t.Errorf("expected empty news payload initially, got %q", got)
	}
	var probe map[string]interface{}
	if err := json.Unmarshal([]byte(tr.Latest("en")), &probe); err != nil {
		t.Errorf("initial payload is not valid JSON: %v", err)
	}

	tr.SetNews("fi", "fi-payload")
	tr.SetNews("en", "en-payload")

	if got := tr.Latest("fi"); got != "fi-payload" {
		t.Errorf("Latest(fi) = %q, want %q", got, "fi-payload")
	}
	if got := tr.Latest("en"); got != "en-payload" {
		t.Errorf("Latest(en) = %q, want %q", got, "en-payload")
	}

	// Overwriting one language must not affect the other.
	tr.SetNews("fi", "fi-updated")
	if got := tr.Latest("en"); got != "en-payload" {
		t.Errorf("Latest(en) changed unexpectedly: %q", got)
	}
}

// TestConcurrentAccess runs against `go test -race`: concurrent SetNews and
// Latest calls must not data race.
func TestConcurrentAccess(t *testing.T) {
	tr := &Tick{}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			tr.SetNews("fi", strings.Repeat("x", i))
		}(i)
		go func() {
			defer wg.Done()
			_ = tr.Latest("fi")
			_ = tr.Latest("en")
		}()
	}
	wg.Wait()
}

// TestJSONMarshalConsistency tests that same data produces same JSON, which
// the change detection in TickNews relies on.
func TestJSONMarshalConsistency(t *testing.T) {
	rss := []domain.RSS{
		{RssTitle: "Test 1", RssLink: "http://example.com"},
		{RssTitle: "Test 2", RssLink: "http://example.com/2"},
	}
	data := map[string]interface{}{"news": rss}

	json1, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	json2, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(json1) != string(json2) {
		t.Error("same data must produce identical JSON strings")
	}
}

// TestRSSFieldPresence verifies that fields clients depend on are marshaled.
func TestRSSFieldPresence(t *testing.T) {
	rss := domain.RSS{RssTitle: "Test Title", RssLink: "http://example.com", Language: "fi"}
	data := map[string]interface{}{"news": []domain.RSS{rss}}

	out, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	for _, want := range []string{"Test Title", "http://example.com", `"language":"fi"`} {
		if !strings.Contains(string(out), want) {
			t.Errorf("expected %q in JSON output %s", want, out)
		}
	}
}

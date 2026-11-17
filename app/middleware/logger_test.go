package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestLoggerWritesValidJSONLine(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "access.log")

	middleware, closer, err := Logger(logPath)
	if err != nil {
		t.Fatalf("Logger: %v", err)
	}
	defer closer.Close()

	e := echo.New()
	c := e.NewContext(
		httptest.NewRequest(http.MethodGet, "/fi/0", nil),
		httptest.NewRecorder(),
	)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	if err := handler(c); err != nil {
		t.Fatalf("handler: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	line := strings.TrimSpace(string(data))
	var entry map[string]string
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v (%s)", err, line)
	}
	for _, key := range []string{"date", "ip", "method", "path", "status", "response-time", "size"} {
		if _, ok := entry[key]; !ok {
			t.Errorf("missing field %q in log entry", key)
		}
	}
	if entry["path"] != "/fi/0" || entry["status"] != "200" {
		t.Errorf("unexpected entry: %+v", entry)
	}
}

func TestLoggerHandlesNextError(t *testing.T) {
	middleware, closer, err := Logger("")
	if err != nil {
		t.Fatalf("Logger: %v", err)
	}
	defer closer.Close()

	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/x", nil), httptest.NewRecorder())
	sentinel := echo.NewHTTPError(http.StatusTeapot, "nope")
	handler := middleware(func(c echo.Context) error { return sentinel })

	if err := handler(c); err != sentinel {
		t.Errorf("expected the handler error to be propagated, got %v", err)
	}
}

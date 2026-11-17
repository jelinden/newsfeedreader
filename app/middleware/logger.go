package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// Logger writes one JSON line per request to the given file (opened once).
// The returned LogCloser must be closed on shutdown.
type LogCloser struct {
	logger *log.Logger
	file   *os.File
}

func (l *LogCloser) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func Logger(logPath string) (echo.MiddlewareFunc, *LogCloser, error) {
	closer := &LogCloser{}
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return nil, nil, err
		}
		closer.file = f
		closer.logger = log.New(f, "", 0)
	} else {
		closer.logger = log.Default()
	}

	// c.RealIP() is used instead of reading X-Real-IP / X-Forwarded-For
	// directly: client-supplied headers must never be trusted verbatim.
	// Configure echo.IPExtractor if running behind a reverse proxy.
	middleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()

			err := next(c)

			stop := time.Now()
			res := c.Response()
			logLine := map[string]string{
				"date":          stop.UTC().Format("2006-01-02 15:04:05"),
				"ip":            c.RealIP(),
				"method":        req.Method,
				"path":          requestPath(req),
				"status":        strconv.Itoa(res.Status),
				"response-time": strconv.FormatFloat(float64(stop.Sub(start).Nanoseconds())/1e6, 'f', 3, 64),
				"size":          strconv.FormatInt(res.Size, 10),
			}
			buf, marshalErr := json.Marshal(logLine)
			if marshalErr != nil {
				log.Println("log marshal error:", marshalErr)
				return err
			}
			closer.logger.Println(string(buf))
			return err
		}
	}
	return middleware, closer, nil
}

func requestPath(req *http.Request) string {
	if req.URL.Path == "" {
		return "/"
	}
	return req.URL.Path
}

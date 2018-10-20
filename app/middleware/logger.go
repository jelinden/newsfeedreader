package middleware

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo"
	"github.com/pquerna/ffjson/ffjson"
)

// Logger is our custom logger
func Logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			res := c.Response()

			remoteAddr := req.RemoteAddr
			if ip := req.Header.Get(echo.HeaderXRealIP); ip != "" {
				remoteAddr = ip
			} else if ip = req.Header.Get(echo.HeaderXForwardedFor); ip != "" {
				remoteAddr = ip
			} else {
				remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
			}

			if err := next(c); err != nil {
				c.Error(err)
			}

			method := req.Method
			path := req.URL.Path
			if path == "" {
				path = "/"
			}
			size := res.Size
			code := strconv.Itoa(res.Status)

			stop := time.Now()
			logLine := map[string]string{
				"date":          time.Now().UTC().Format("2006/01/02 15:04:05"),
				"ip":            remoteAddr,
				"method":        method,
				"path":          path,
				"status":        code,
				"response-time": fmt.Sprintf("%.3fms", float64(stop.Sub(start).Nanoseconds())/(1e6*1.0)),
				"size":          fmt.Sprintf("%v", size),
			}
			buf, _ := ffjson.Marshal(&logLine)

			f, err := os.OpenFile("access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				log.Printf("error opening file: %v", err)
			}
			defer f.Close()
			logger := log.New(f, "", 0)
			logger.SetOutput(f)

			logger.Println(string(buf))
			return nil
		}
	}
}

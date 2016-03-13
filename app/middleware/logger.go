package middleware

import (
	"github.com/labstack/echo"
	"log"
	"net"
	"os"
	"time"
)

func Logger() echo.MiddlewareFunc {
	return func(next echo.Handler) echo.Handler {
		return echo.HandlerFunc(func(c echo.Context) error {
			req := c.Request()
			res := c.Response()

			remoteAddr := req.RemoteAddress()
			if ip := req.Header().Get(echo.XRealIP); ip != "" {
				remoteAddr = ip
			} else if ip = req.Header().Get(echo.XForwardedFor); ip != "" {
				remoteAddr = ip
			} else {
				remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
			}

			start := time.Now()
			if err := next.Handle(c); err != nil {
				c.Error(err)
			}
			stop := time.Now()
			method := req.Method()
			path := req.URL().Path()
			if path == "" {
				path = "/"
			}
			size := res.Size()
			code := res.Status()

			f, err := os.OpenFile("access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				log.Printf("error opening file: %v", err)
			}
			defer f.Close()
			logger := log.New(f, "", log.LstdFlags)
			logger.SetOutput(f)

			logger.Println(remoteAddr, method, path, code, stop.Sub(start), size)
			return nil
		})
	}
}

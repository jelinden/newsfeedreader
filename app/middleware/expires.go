package middleware

import (
	"fmt"
	"github.com/labstack/echo"
	"net/http"
	"time"
)

var secondsInAYear = 365 * 24 * 60 * 60

func Expires() echo.MiddlewareFunc {
	return func(next echo.Handler) echo.Handler {
		return echo.HandlerFunc(func(c echo.Context) error {
			c.Response().Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public, must-revalidate, proxy-revalidate", secondsInAYear))
			c.Response().Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
			c.Response().Header().Set("Expires", time.Now().AddDate(1, 0, 0).Format(http.TimeFormat))
			if err := next.Handle(c); err != nil {
				fmt.Println("error", err)
				c.Error(err)
			}
			return nil
		})
	}
}

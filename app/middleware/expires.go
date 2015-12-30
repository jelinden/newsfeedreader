package middleware

import (
	"fmt"
	"github.com/labstack/echo"
	"net/http"
	"time"
)

var secondsInAYear = 365 * 24 * 60 * 60

func Expires() echo.Middleware {
	return func(c *echo.Context) error {
		c.Response().Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public, must-revalidate, proxy-revalidate", secondsInAYear))
		c.Response().Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
		c.Response().Header().Set("Expires", time.Now().AddDate(1, 0, 0).Format(http.TimeFormat))
		return nil
	}
}

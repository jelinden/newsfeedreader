package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

var secondsInAYear = 365 * 24 * 60 * 60

func Expires() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public, must-revalidate, proxy-revalidate", secondsInAYear))
			c.Response().Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
			c.Response().Header().Set("Expires", time.Now().AddDate(1, 0, 0).Format(http.TimeFormat))
			if err := next(c); err != nil {
				log.Println("error", err)
				c.Error(err)
			}
			return nil
		}
	}
}

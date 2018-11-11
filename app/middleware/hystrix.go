package middleware

import (
	"log"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/labstack/echo"
)

func Hystrix() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			hystrix.Do(c.Request().RequestURI, func() error {
				return next(c)
			}, func(err error) error {
				log.Println(err)
				return err
			})
			return nil
		}
	}
}

package middleware

import (
	"log"
	"strings"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/labstack/echo"
)

func Hystrix() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			hystrix.Do(c.Request().RequestURI, func() error {
				return next(c)
			}, func(err error) error {
				log.Println("hystrix ERROR", err.Error())
				if strings.Contains(err.Error(), "code=404") {
					return c.HTML(404, `<html><head></head><body>Oops, didn't find anything (404)<br/>`+
						`<a href="https://www.uutispuro.fi/en">Uutispuro.fi</a></body>`)
				} else if strings.Contains(err.Error(), "code=50") {
					return c.HTML(404, `<html><head></head><body>Oops, technical difficulties, please try again after a while (50x)<br/>`+
						`<a href="https://www.uutispuro.fi/en">Uutispuro.fi</a></body>`)
				}
				return err
			})
			return nil
		}
	}
}

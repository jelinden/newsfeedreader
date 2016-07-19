package util

import (
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/engine"
)

type CookieUtil struct {
}

func NewCookieUtil() *CookieUtil {
	return &CookieUtil{}
}

type (
	responseWriter struct {
		engine.Response
		engine.Header
		io.Writer
	}
)

func (c *CookieUtil) SetCookie(name string, value string, context echo.Context) {
	expire := time.Now().AddDate(1, 0, 1)
	cookie := http.Cookie{name, value, "/", ".uutispuro.fi", expire, expire.Format(time.UnixDate), 41472000, false, false, name + "=" + value, []string{name + "=" + value}}
	context.Response().Header().Add("Set-Cookie", cookie.String())
}

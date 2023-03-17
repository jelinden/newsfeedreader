package util

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type CookieUtil struct {
}

func NewCookieUtil() *CookieUtil {
	return &CookieUtil{}
}

func (c *CookieUtil) SetCookie(name string, value string, context echo.Context) {
	expire := time.Now().AddDate(1, 0, 1)
	nameValuePair := name + "=" + value
	cookie := http.Cookie{
		Name:       name,
		Value:      value,
		Path:       "/",
		Domain:     ".uutispuro.fi",
		Expires:    expire,
		RawExpires: expire.Format(time.UnixDate),
		MaxAge:     41472000,
		Secure:     false,
		HttpOnly:   false,
		SameSite:   1,
		Raw:        nameValuePair,
		Unparsed:   []string{nameValuePair},
	}
	context.Response().Header().Add("Set-Cookie", cookie.String())
}

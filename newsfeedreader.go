package main

import (
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/jelinden/newsfeedreader/service"
	"github.com/jelinden/newsfeedreader/util"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"github.com/thoas/stats"
)

type (
	Template struct {
		templates *template.Template
	}
	Application struct {
		Sessions   *service.Sessions
		CookieUtil *util.CookieUtil
	}
)

func NewApplication() *Application {
	return &Application{}
}

func (a *Application) Init() {
	a.Sessions = service.NewSessions()
	a.CookieUtil = util.NewCookieUtil()
	a.Sessions.Init()
}

func (a *Application) Close() {
	a.Sessions.Close()
}

func main() {
	app := NewApplication()
	app.Init()
	defer app.Close()

	e := echo.New()

	e.Get("/", func(c *echo.Context) error {
		lang, err := c.Request().Cookie("uutispuroLang")
		if err != nil {
			log.Println(err)
			c.Redirect(302, "/en")
		} else if lang.Value == "fi" {
			c.Redirect(302, "/fi")
		} else {
			c.Redirect(302, "/en")
		}
		return nil
	})
	e.Favicon("public/favicon.ico")
	e.Use(mw.Logger())
	e.Use(mw.Recover())
	e.Use(mw.Gzip())
	e.StripTrailingSlash()
	s := stats.New()
	e.Use(s.Handler)

	e.Get("/stats", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, s.Data())
	})

	t := &Template{
		templates: template.Must(template.ParseFiles("public/html/index_fi.html")),
	}
	e.SetRenderer(t)
	e.Get("/fi", func(c *echo.Context) error {
		return app.renderer("index_fi", c)
	})

	e.Run(":1300")
}

func (a *Application) renderer(page string, c *echo.Context) error {
	return c.Render(http.StatusOK, page, a.Sessions.FetchRssItems("fi"))
}

func (t *Template) Render(w io.Writer, name string, data interface{}) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

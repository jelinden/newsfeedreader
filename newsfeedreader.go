package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/jelinden/newsfeedreader/app/middleware"
	"github.com/jelinden/newsfeedreader/app/render"
	"github.com/jelinden/newsfeedreader/app/routes"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/tick"
	"github.com/jelinden/newsfeedreader/app/util"
	"github.com/labstack/echo/v4"
	mw "github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/websocket"
)

type Application struct {
	Mongo      *service.Mongo
	CookieUtil *util.CookieUtil
	Tick       *tick.Tick
	Render     *render.Render
}

var app *Application

const secondsInAYear = 365 * 24 * 60 * 60

func (a *Application) Start() {
	a.Mongo = service.NewMongo(os.Getenv("MONGO_URL"))
	a.CookieUtil = util.NewCookieUtil()
	a.Tick = tick.NewTick(a.Mongo)
	a.Render = render.NewRender(a.Mongo)
}

func (a *Application) Close() {
	log.Println("closing up")
	a.Mongo.Close()
}

func main() {
	app = &Application{}
	app.Start()
	defer app.Close()

	e := echo.New()
	e.Use(mw.RemoveTrailingSlashWithConfig(mw.TrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}))

	e.Use(mw.Recover())

	go app.Tick.TickNews("fi")
	go app.Tick.TickNews("en")

	paths := e.Group("/")
	paths.Use(mw.Gzip())
	paths.Use(middleware.Logger())
	paths.GET("", routes.Root)
	paths.GET("fi", routes.FiRoot(app.Render))
	paths.GET("en", routes.EnRoot(app.Render))
	paths.GET("fi/login", routes.Login(app.Render, "fi"))
	paths.GET("en/login", routes.Login(app.Render, "en"))
	paths.GET("fi/:page", routes.FiRootPaged(app.Render))
	paths.GET("en/:page", routes.EnRootPaged(app.Render))
	paths.GET("fi/search", routes.FiSearch(app.Render))
	paths.GET("en/search", routes.EnSearch(app.Render))
	paths.GET("fi/search/:page", routes.FiSearchPaged(app.Render))
	paths.GET("en/search/:page", routes.EnSearchPaged(app.Render))

	paths.GET("fi/category/:category/:page", routes.FiCategory(app.Render))
	paths.GET("en/category/:category/:page", routes.EnCategory(app.Render))
	paths.GET("fi/source/:source/:page", routes.FiSource(app.Render))
	paths.GET("en/source/:source/:page", routes.EnSource(app.Render))
	paths.GET("fi/category/:category", redirect)
	paths.GET("en/category/:category", redirect)
	paths.GET("fi/source/:source", redirect)
	paths.GET("en/source/:source", redirect)

	paths.GET("uutiset/fi", func(c echo.Context) error {
		c.Response().Header().Set("Location", "/fi/0")
		return c.NoContent(http.StatusMovedPermanently)
	})
	paths.GET("uutiset/en", func(c echo.Context) error {
		c.Response().Header().Set("Location", "/en/0")
		return c.NoContent(http.StatusMovedPermanently)
	})

	paths.GET("api/click/:id", routes.Click(app.Mongo))

	paths.GET("public/:filePath/:fileName", static)
	paths.File("favicon.ico", "public/img/favicon.ico")
	paths.File("sitemap.xml", "public/sitemap.xml")
	paths.File("robots.txt", "public/robots.txt")
	paths.File("ads.txt", "public/ads.txt")
	paths.File("public/sitemap.xml", "public/sitemap.xml")
	paths.GET("serviceworker.js", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/javascript")
		return c.File("public/js/serviceworker.js")
	})

	paths.GET("api/news", routes.News)
	paths.GET("ws/:channel", ws)

	log.Fatal(e.Start(":1300"))
}

func redirect(c echo.Context) error {
	c.Response().Header().Set("Location", c.Request().URL.Path+"/0")
	return c.NoContent(http.StatusMovedPermanently)
}

func static(c echo.Context) error {
	filePath := c.Param("filePath")
	if filePath == "js" || filePath == "img" || filePath == "css" {
		c.Response().Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public, must-revalidate, proxy-revalidate", secondsInAYear))
		c.Response().Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
		c.Response().Header().Set("Expires", time.Now().AddDate(1, 0, 0).Format(http.TimeFormat))
		return c.File(path.Join("public", c.Param("filePath"), c.Param("fileName")))
	}
	return c.JSONBlob(http.StatusNotFound, nil)
}

func ws(c echo.Context) error {
	channel := c.Param("channel")
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if channel == "fi" {
				websocket.Message.Send(ws, app.Tick.NewsFi)
			}
			if channel == "en" {
				websocket.Message.Send(ws, app.Tick.NewsEn)
			}
			time.Sleep(15 * time.Second)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

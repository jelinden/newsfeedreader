package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/googollee/go-socket.io"
	"github.com/jelinden/newsfeedreader/app/middleware"
	"github.com/jelinden/newsfeedreader/app/render"
	"github.com/jelinden/newsfeedreader/app/routes"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/tick"
	"github.com/jelinden/newsfeedreader/app/util"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine"
	"github.com/labstack/echo/engine/standard"
	mw "github.com/labstack/echo/middleware"
	"golang.org/x/net/http2"
)

type (
	Application struct {
		Mongo      *service.Mongo
		CookieUtil *util.CookieUtil
		Tick       *tick.Tick
		Render     *render.Render
	}
)

func (a *Application) Start() {
	a.Mongo = service.NewMongo()
	a.CookieUtil = util.NewCookieUtil()
	a.Tick = tick.NewTick(a.Mongo)
	a.Render = render.NewRender(a.Mongo)
}

func (a *Application) Close() {
	log.Println("closing up")
	a.Mongo.Close()
}

func main() {
	app := &Application{}
	app.Start()
	e := echo.New()
	e.Use(mw.RemoveTrailingSlashWithConfig(mw.TrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}))
	e.Use(mw.Gzip())
	e.Use(mw.Recover())
	e.Use(middleware.Logger())

	defer app.Close()
	go app.Tick.TickNews("fi")
	go app.Tick.TickNews("en")
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	go app.Tick.TickEmit(server)
	server.On("connection", func(so socketio.Socket) {
		ref := so.Request().Referer()
		var referer string
		if strings.Contains(ref, "https") {
			referer = strings.Replace(ref, "https://", "", 1)
		} else {
			referer = strings.Replace(ref, "http://", "", 1)
		}

		pathArr := strings.Split(referer, "/")
		var path = ""
		if len(pathArr) > 1 {
			path = pathArr[1]
		}

		log.Println("connecting to", path)
		so.Join(path)

		so.On("disconnection", func() {
			so.Leave(path)
		})
	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("socketio error:", err)
	})

	e.GET("/", routes.Root())
	e.GET("/fi", routes.FiRoot(app.Render))
	e.GET("/en", routes.EnRoot(app.Render))
	e.GET("/fi/login", routes.Login(app.Render, "fi"))
	e.GET("/en/login", routes.Login(app.Render, "en"))
	e.GET("/fi/:page", routes.FiRootPaged(app.Render))
	e.GET("/en/:page", routes.EnRootPaged(app.Render))
	e.GET("/fi/search", routes.FiSearch(app.Render))
	e.GET("/en/search", routes.EnSearch(app.Render))
	e.GET("/fi/search/:page", routes.FiSearchPaged(app.Render))
	e.GET("/en/search/:page", routes.EnSearchPaged(app.Render))
	e.GET("/fi/category/:category/:page", routes.FiCategory(app.Render))
	e.GET("/en/category/:category/:page", routes.EnCategory(app.Render))
	e.GET("/fi/source/:source/:page", routes.FiSource(app.Render))
	e.GET("/en/source/:source/:page", routes.EnSource(app.Render))
	e.GET("/api/click/:id", routes.Click(app.Mongo))

	var secondsInAYear = 365 * 24 * 60 * 60
	e.GET("/public*", func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public, must-revalidate, proxy-revalidate", secondsInAYear))
		c.Response().Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
		c.Response().Header().Set("Expires", time.Now().AddDate(1, 0, 0).Format(http.TimeFormat))
		return c.File(path.Join("public", c.P(0)))
	})
	e.File("/favicon.ico", "public/img/favicon.ico")
	e.GET("/serviceworker.js", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/javascript")
		return c.File("public/js/serviceworker.js")
	})
	http.Handle("/socket.io/", server)

	// hook echo with http handler
	std := standard.WithConfig(engine.Config{})
	std.SetHandler(e)
	http2.ConfigureServer(std.Server, &http2.Server{})
	http.Handle("/", std)
	log.Println("Starting up server at port 1300")
	log.Fatal(http.ListenAndServe(":1300", nil))
}

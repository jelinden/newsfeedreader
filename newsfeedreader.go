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
		Nats       *middleware.Nats
	}
)

func (a *Application) Start() {
	a.Mongo = service.NewMongo()
	a.CookieUtil = util.NewCookieUtil()
	a.Tick = tick.NewTick(a.Mongo)
	a.Render = render.NewRender(a.Mongo)
	a.Nats = middleware.NewNats()
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
	e.Use(middleware.NatsHandler(app.Nats))

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

	e.GET("/", middleware.Root())
	e.GET("/fi", middleware.FiRoot(app.Render))
	e.GET("/en", middleware.EnRoot(app.Render))
	e.GET("/fi/login", middleware.Login(app.Render, "fi"))
	e.GET("/en/login", middleware.Login(app.Render, "en"))
	e.GET("/fi/:page", middleware.FiRootPaged(app.Render))
	e.GET("/en/:page", middleware.EnRootPaged(app.Render))
	e.GET("/fi/search", middleware.FiSearch(app.Render))
	e.GET("/en/search", middleware.EnSearch(app.Render))
	e.GET("/fi/search/:page", middleware.FiSearchPaged(app.Render))
	e.GET("/en/search/:page", middleware.EnSearchPaged(app.Render))
	e.GET("/fi/category/:category/:page", middleware.FiCategory(app.Render))
	e.GET("/en/category/:category/:page", middleware.EnCategory(app.Render))
	e.GET("/fi/source/:source/:page", middleware.FiSource(app.Render))
	e.GET("/en/source/:source/:page", middleware.EnSource(app.Render))
	e.GET("/api/click/:id", middleware.Click(app.Mongo))

	var secondsInAYear = 365 * 24 * 60 * 60
	e.GET("/public*", func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public, must-revalidate, proxy-revalidate", secondsInAYear))
		c.Response().Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
		c.Response().Header().Set("Expires", time.Now().AddDate(1, 0, 0).Format(http.TimeFormat))
		return c.File(path.Join("public", c.P(0)))
	})
	e.File("/favicon.ico", "public/favicon.ico")

	http.Handle("/socket.io/", server)

	// hook echo with http handler
	std := standard.WithConfig(engine.Config{})
	std.SetHandler(e)
	//http2.ConfigureServer(std.Server, nil)
	http2.ConfigureServer(std.Server, &http2.Server{})
	http.Handle("/", std)
	log.Fatal(http.ListenAndServe(":1300", nil))
}

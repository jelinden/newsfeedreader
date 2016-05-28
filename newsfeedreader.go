package main

import (
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
	"log"
	"net/http"
	"strings"
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

func NewApplication() *Application {
	return &Application{}
}

func (a *Application) Init() {
	a.Mongo = service.NewMongo()
	a.CookieUtil = util.NewCookieUtil()
	a.Tick = tick.NewTick(a.Mongo)
	a.Render = render.NewRender(a.Mongo)
	a.Nats = middleware.NewNats()
}

func (a *Application) Close() {
	a.Mongo.Close()
}

func main() {
	app := NewApplication()
	app.Init()
	e := echo.New()
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

	e.Get("/", middleware.Root())
	e.Get("/fi", middleware.FiRoot(app.Render))
	e.Get("/en", middleware.EnRoot(app.Render))
	e.Get("/fi/:page", middleware.FiRootPaged(app.Render))
	e.Get("/en/:page", middleware.EnRootPaged(app.Render))
	e.Get("/fi/search", middleware.FiSearch(app.Render))
	e.Get("/en/search", middleware.EnSearch(app.Render))
	e.Get("/fi/search/:page", middleware.FiSearchPaged(app.Render))
	e.Get("/en/search/:page", middleware.EnSearchPaged(app.Render))
	e.Get("/fi/category/:category/:page", middleware.FiCategory(app.Render))
	e.Get("/en/category/:category/:page", middleware.EnCategory(app.Render))
	e.Get("/fi/source/:source/:page", middleware.FiSource(app.Render))
	e.Get("/en/source/:source/:page", middleware.EnSource(app.Render))
	e.Get("/api/click/:id", middleware.Click(app.Mongo))

	e.Static("/public", "public")
	e.File("/favicon.ico", "public/favicon.ico")

	http.Handle("/socket.io/", server)

	// hook echo with http handler
	std := standard.WithConfig(engine.Config{})
	std.SetHandler(e)
	http2.ConfigureServer(std.Server, nil)
	http.Handle("/", std)
	log.Fatal(http.ListenAndServe(":1300", nil))
}

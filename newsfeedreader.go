package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/jelinden/newsfeedreader/app/middleware"
	"github.com/jelinden/newsfeedreader/app/render"
	"github.com/jelinden/newsfeedreader/app/routes"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/tick"
	"github.com/jelinden/newsfeedreader/app/util"
	"github.com/kabukky/httpscerts"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
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
	a.Mongo = service.NewMongo()
	a.CookieUtil = util.NewCookieUtil()
	a.Tick = tick.NewTick(a.Mongo)
	a.Render = render.NewRender(a.Mongo)
}

func (a *Application) Close() {
	log.Println("closing up")
	a.Mongo.Close()
}

func environment() string {
	env := flag.String("env", "", "-env local")
	flag.Parse()
	if *env != "local" && *env != "prod" {
		fmt.Println("------\nEnvironment flag missing (-env local|prod)\n------")
		os.Exit(-1)
	}
	return *env
}

func main() {
	env := environment()
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
	paths.Use(middleware.Hystrix())
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
	paths.GET("api/click/:id", routes.Click(app.Mongo))

	paths.GET("public/:filePath/:fileName", static)
	paths.File("favicon.ico", "public/img/favicon.ico")
	paths.File("sitemap.xml", "public/sitemap.xml")
	paths.File("public/sitemap.xml", "public/sitemap.xml")
	paths.GET("serviceworker.js", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/javascript")
		return c.File("public/js/serviceworker.js")
	})

	paths.GET("ws/:channel", ws)

	err := httpscerts.Check("cert.pem", "key.pem")
	if err != nil {
		err = httpscerts.Generate("cert.pem", "key.pem", "localdev.uutispuro.fi:443")
		if err != nil {
			log.Fatal("Error: Couldn't create https certs.")
		}
	}
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	go http.ListenAndServe(net.JoinHostPort("0.0.0.0", "8181"), hystrixStreamHandler)

	if env == "prod" {
		log.Fatal(e.Start(":1300"))
	}
	log.Fatal(e.TLSServer.ListenAndServeTLS("cert.pem", "key.pem"))
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
			time.Sleep(10 * time.Second)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

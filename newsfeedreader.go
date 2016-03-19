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
	"regexp"
	"strconv"
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

	e.Get("/", echo.HandlerFunc(func(c echo.Context) error {
		lang := c.Request().Header().Get("Accept-Language")
		if lang == "" {
			return c.Redirect(http.StatusTemporaryRedirect, "/fi")
		} else if strings.Contains(strings.Split(lang, ",")[0], "en") {
			return c.Redirect(http.StatusTemporaryRedirect, "/en")
		}
		return c.Redirect(http.StatusTemporaryRedirect, "/fi")
	}))
	e.Get("/fi", echo.HandlerFunc(func(c echo.Context) error {
		return app.Render.RenderIndex("index_fi", "fi", 0, c, http.StatusOK)
	}))
	e.Get("/en", echo.HandlerFunc(func(c echo.Context) error {
		return app.Render.RenderIndex("index_en", "en", 0, c, http.StatusOK)
	}))
	e.Get("/fi/:page", echo.HandlerFunc(func(c echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderIndex("index_fi", "fi", page, c, http.StatusOK)
			}
		}
		return app.Render.RenderIndex("index_fi", "fi", 0, c, http.StatusBadRequest)
	}))
	e.Get("/en/:page", echo.HandlerFunc(func(c echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderIndex("index_en", "en", page, c, http.StatusOK)
			}
		}
		return app.Render.RenderIndex("index_en", "en", 0, c, http.StatusBadRequest)
	}))
	e.Get("/fi/search", echo.HandlerFunc(func(c echo.Context) error {
		return app.Render.RenderSearch("search_fi", "fi", app.validateAndCorrectifySearchTerm(c.Form("q")), 0, c, http.StatusOK)
	}))
	e.Get("/en/search", echo.HandlerFunc(func(c echo.Context) error {
		return app.Render.RenderSearch("search_en", "en", app.validateAndCorrectifySearchTerm(c.Form("q")), 0, c, http.StatusOK)
	}))
	e.Get("/fi/search/:page", echo.HandlerFunc(func(c echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderSearch("search_fi", "fi", app.validateAndCorrectifySearchTerm(c.Form("q")), page, c, http.StatusOK)
			}
		}
		return app.Render.RenderSearch("search_fi", "fi", app.validateAndCorrectifySearchTerm(c.Form("q")), 0, c, http.StatusOK)
	}))
	e.Get("/fi/category/:category/:page", echo.HandlerFunc(func(c echo.Context) error {
		category := util.ToUpper(c.P(0))
		if page, err := strconv.Atoi(c.P(1)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderByCategory("category_fi", "fi", app.validateAndCorrectifySearchTerm(category), page, c, http.StatusOK)
			}
		}
		return app.Render.RenderByCategory("category_fi", "fi", app.validateAndCorrectifySearchTerm(category), 0, c, http.StatusOK)
	}))
	e.Get("/en/category/:category/:page", echo.HandlerFunc(func(c echo.Context) error {
		category := util.ToUpper(c.P(0))
		if page, err := strconv.Atoi(c.P(1)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderByCategory("category_en", "en", app.validateAndCorrectifySearchTerm(category), page, c, http.StatusOK)
			}
		}
		return app.Render.RenderByCategory("category_en", "en", app.validateAndCorrectifySearchTerm(category), 0, c, http.StatusOK)
	}))
	e.Get("/fi/source/:source/:page", echo.HandlerFunc(func(c echo.Context) error {
		category := util.ToUpper(c.P(0))
		if page, err := strconv.Atoi(c.P(1)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderBySource("source_fi", "fi", app.validateAndCorrectifySearchTerm(category), page, c, http.StatusOK)
			}
		}
		return app.Render.RenderBySource("source_fi", "fi", app.validateAndCorrectifySearchTerm(category), 0, c, http.StatusOK)
	}))
	e.Get("/en/source/:source/:page", echo.HandlerFunc(func(c echo.Context) error {
		category := util.ToUpper(c.P(0))
		if page, err := strconv.Atoi(c.P(1)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderBySource("source_en", "en", app.validateAndCorrectifySearchTerm(category), page, c, http.StatusOK)
			}
		}
		return app.Render.RenderBySource("source_en", "en", app.validateAndCorrectifySearchTerm(category), 0, c, http.StatusOK)
	}))
	e.Get("/en/search/:page", echo.HandlerFunc(func(c echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderSearch("search_en", "en", app.validateAndCorrectifySearchTerm(c.Form("q")), page, c, http.StatusOK)
			}
		}
		return app.Render.RenderSearch("search_en", "en", app.validateAndCorrectifySearchTerm(c.Form("q")), 0, c, http.StatusOK)
	}))

	e.Get("/api/click/:id", echo.HandlerFunc(func(c echo.Context) error {
		app.Mongo.SaveClick(app.validateAndCorrectifySearchTerm(c.P(0)))
		return c.NoContent(http.StatusOK)
	}))
	e.Static("/public", "public")
	e.File("/favicon.ico", "public/favicon.ico")

	http.Handle("/socket.io/", server)

	// hook echo with http handler
	std := standard.NewFromConfig(engine.Config{})
	std.SetHandler(e)
	http2.ConfigureServer(std.Server, nil)
	http.Handle("/", std)
	log.Fatal(http.ListenAndServe(":1300", nil))
}

func (a *Application) validateAndCorrectifySearchTerm(searchString string) string {
	r, _ := regexp.Compile("[^-a-zåäöA-ZÅÄÖ0-9 ]+")
	return string(r.ReplaceAll([]byte(searchString), []byte(""))[:])
}

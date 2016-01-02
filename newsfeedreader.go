package main

import (
	"github.com/googollee/go-socket.io"
	"github.com/jelinden/newsfeedreader/app/middleware"
	"github.com/jelinden/newsfeedreader/app/render"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/tick"
	"github.com/jelinden/newsfeedreader/app/util"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
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
}

func (a *Application) Close() {
	a.Mongo.Close()
}

func main() {
	app := NewApplication()
	app.Init()
	e := echo.New()
	e.Use(mw.Gzip())
	e.Use(middleware.Logger())
	e.Favicon("./public/favicon.ico")
	e.Hook(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		l := len(path) - 1
		if path != "/" && path[l] == '/' {
			r.URL.Path = path[:l]
		}
	})
	defer app.Close()
	go app.Tick.TickNews("fi")
	go app.Tick.TickNews("en")
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	go app.Tick.TickEmit(server)
	server.On("connection", func(so socketio.Socket) {
		referer := strings.Replace(so.Request().Referer(), "http://", "", 1)
		pathArr := strings.Split(referer, "/")
		path := pathArr[1]

		log.Println("connecting to", path)
		so.Join(path)

		so.On("disconnection", func() {
			so.Leave(path)
		})
	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("socketio error:", err)
	})

	e.Get("/", func(c *echo.Context) error {
		lang := c.Request().Header.Get("Accept-Language")
		if lang == "" {
			return c.Redirect(http.StatusTemporaryRedirect, "/fi")
		} else if strings.Contains(strings.Split(lang, ",")[0], "en") {
			return c.Redirect(http.StatusTemporaryRedirect, "/en")
		}
		return c.Redirect(http.StatusTemporaryRedirect, "/fi")
	})
	e.Get("/fi", func(c *echo.Context) error {
		return app.Render.RenderIndex("index_fi", "fi", 0, c, http.StatusOK)
	})
	e.Get("/en", func(c *echo.Context) error {
		return app.Render.RenderIndex("index_en", "en", 0, c, http.StatusOK)
	})
	e.Get("/fi/:page", func(c *echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderIndex("index_fi", "fi", page, c, http.StatusOK)
			}
		}
		return app.Render.RenderIndex("index_fi", "fi", 0, c, http.StatusBadRequest)
	})
	e.Get("/en/:page", func(c *echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderIndex("index_en", "en", page, c, http.StatusOK)
			}
		}
		return app.Render.RenderIndex("index_en", "en", 0, c, http.StatusBadRequest)
	})
	e.Get("/fi/search", func(c *echo.Context) error {
		return app.Render.RenderSearch("search_fi", "fi", app.validateAndCorrectifySearchTerm(c.Form("q")), 0, c, http.StatusOK)
	})
	e.Get("/en/search", func(c *echo.Context) error {
		return app.Render.RenderSearch("search_en", "en", app.validateAndCorrectifySearchTerm(c.Form("q")), 0, c, http.StatusOK)
	})
	e.Get("/fi/search/:page", func(c *echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderSearch("search_fi", "fi", app.validateAndCorrectifySearchTerm(c.Form("q")), page, c, http.StatusOK)
			}
		}
		return app.Render.RenderSearch("search_fi", "fi", app.validateAndCorrectifySearchTerm(c.Form("q")), 0, c, http.StatusOK)
	})
	e.Get("/en/search/:page", func(c *echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return app.Render.RenderSearch("search_en", "en", app.validateAndCorrectifySearchTerm(c.Form("q")), page, c, http.StatusOK)
			}
		}
		return app.Render.RenderSearch("search_en", "en", app.validateAndCorrectifySearchTerm(c.Form("q")), 0, c, http.StatusOK)
	})
	s := e.Group("/public")
	s.Use(middleware.Expires())
	s.ServeDir("", "./public")
	http.Handle("/socket.io/", server)
	// hook echo with http handler
	http.Handle("/", e)
	log.Fatal(http.ListenAndServe(":1300", nil))
}

func (a *Application) validateAndCorrectifySearchTerm(searchString string) string {
	r, _ := regexp.Compile("[^-a-zåäöA-ZÅÄÖ0-9 ]+")
	return string(r.ReplaceAll([]byte(searchString), []byte(""))[:])
}

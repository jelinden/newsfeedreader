package main

import (
	"github.com/googollee/go-socket.io"
	"github.com/jelinden/newsfeedreader/domain"
	"github.com/jelinden/newsfeedreader/middleware"
	"github.com/jelinden/newsfeedreader/service"
	"github.com/jelinden/newsfeedreader/tick"
	"github.com/jelinden/newsfeedreader/util"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"github.com/rsniezynski/go-asset-helper"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type (
	Template struct {
		templates *template.Template
	}
	Application struct {
		Mongo      *service.Mongo
		CookieUtil *util.CookieUtil
		Tick       *tick.Tick
	}
)

func NewApplication() *Application {
	return &Application{}
}

func (a *Application) Init() {
	a.Mongo = service.NewMongo()
	a.CookieUtil = util.NewCookieUtil()
	a.Tick = tick.NewTick(a.Mongo)
}

func (a *Application) Close() {
	a.Mongo.Close()
}

func main() {
	app := NewApplication()
	app.Init()
	e := echo.New()
	e.Use(mw.Gzip())
	e.Use(mw.Logger())
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
		log.Println("error:", err)
	})
	static, _ := asset.NewStatic("", "./manifest.json")
	t := &Template{
		templates: template.Must(template.New("").Funcs(static.FuncMap()).Funcs(template.FuncMap{
			"minus": func(a, b int) int {
				return a - b
			},
			"add": func(a, b int) int {
				return a + b
			},
		}).ParseFiles("public/html/index_fi.html", "public/html/index_en.html")),
	}
	e.SetRenderer(t)
	e.Get("/fi", func(c *echo.Context) error {
		return app.renderer("index_fi", "fi", 0, c, http.StatusOK)
	})
	e.Get("/en", func(c *echo.Context) error {
		return app.renderer("index_en", "en", 0, c, http.StatusOK)
	})
	e.Get("/fi/:page", func(c *echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return app.renderer("index_fi", "fi", page, c, http.StatusOK)
			}
		}
		return app.renderer("index_fi", "fi", 0, c, http.StatusBadRequest)
	})
	e.Get("/en/:page", func(c *echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return app.renderer("index_en", "en", page, c, http.StatusOK)
			}
		}
		return app.renderer("index_en", "en", 0, c, http.StatusBadRequest)
	})
	s := e.Group("/public")
	s.Use(middleware.Expires())
	s.ServeDir("", "./public")
	http.Handle("/socket.io/", server)
	// hook echo with http handler
	http.Handle("/", e)
	log.Fatal(http.ListenAndServe(":1300", nil))
}

func (a *Application) renderer(name string, lang string, page int, c *echo.Context, statusCode int) error {
	news := &domain.News{Page: page, RSS: a.Mongo.FetchRssItems(lang, page, 30)}
	return c.Render(statusCode, name, news)
}

// Render HTML
func (t *Template) Render(w io.Writer, name string, data interface{}) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

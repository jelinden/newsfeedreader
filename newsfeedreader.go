package main

import (
	"fmt"
	"github.com/googollee/go-socket.io"
	"github.com/jelinden/newsfeedreader/service"
	"github.com/jelinden/newsfeedreader/tick"
	"github.com/jelinden/newsfeedreader/util"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"html/template"
	"io"
	"log"
	"net/http"
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

		fmt.Println("connecting to", path)
		so.Join(path)

		so.On("disconnection", func() {
			so.Leave(path)
		})
	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	t := &Template{
		templates: template.Must(template.ParseFiles("public/html/index_fi.html", "public/html/index_en.html")),
	}
	e.SetRenderer(t)
	e.Get("/fi", func(c *echo.Context) error {
		return app.renderer("index_fi", "fi", c)
	})
	e.Get("/en", func(c *echo.Context) error {
		return app.renderer("index_en", "en", c)
	})
	e.Get("/fi/:page", func(c *echo.Context) error {
		fmt.Println("page", c.P(0))
		return app.renderer("index_fi", "fi", c)
	})
	e.Get("/en/:page", func(c *echo.Context) error {
		fmt.Println("page", c.P(0))
		return app.renderer("index_en", "en", c)
	})
	e.ServeDir("/public", "./public")
	http.Handle("/socket.io/", server)
	// hook echo with http handler
	http.Handle("/", e)
	log.Fatal(http.ListenAndServe(":1300", nil))
}

func (a *Application) renderer(name string, lang string, c *echo.Context) error {
	return c.Render(http.StatusOK, name, map[string]interface{}{"news": a.Mongo.FetchRssItems(lang, 0, 30)})
}

// Render HTML
func (t *Template) Render(w io.Writer, name string, data interface{}) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

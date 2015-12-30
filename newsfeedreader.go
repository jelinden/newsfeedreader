package main

import (
	"bytes"
	"github.com/googollee/go-socket.io"
	"github.com/jelinden/newsfeedreader/app/domain"
	"github.com/jelinden/newsfeedreader/app/middleware"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/tick"
	"github.com/jelinden/newsfeedreader/app/util"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"github.com/rsniezynski/go-asset-helper"
	"github.com/wunderlist/ttlcache"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
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

var t = &Template{}
var cache = ttlcache.NewCache(time.Minute)

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
	t = &Template{
		templates: template.Must(template.New("").Funcs(static.FuncMap()).Funcs(template.FuncMap{
			"minus": func(a, b int) int {
				return a - b
			},
			"add": func(a, b int) int {
				return a + b
			},
		}).ParseFiles("public/html/index_fi.html", "public/html/index_en.html")),
	}
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
	pString := strconv.Itoa(page)
	key := name + "_" + pString
	value, exists := cache.Get(key)
	if exists {
		log.Println("found from cache", cache.Count())
		return a.Render(http.StatusOK, name, []byte(value), c)
	} else {
		var buf bytes.Buffer
		err := t.templates.ExecuteTemplate(&buf, name, &domain.News{Page: page, RSS: a.Mongo.FetchRssItems(lang, page, 30)})
		if err != nil {
			log.Println("rendering page", name, "failed.", err.Error())
			return err
		}
		cache.Set(key, buf.String())
		log.Println("cache count after add", cache.Count())
		return a.Render(http.StatusOK, name, buf.Bytes(), c)
	}
}

func (a *Application) Render(code int, name string, data []byte, c *echo.Context) (err error) {
	c.Response().Header().Set(echo.ContentType, echo.TextHTMLCharsetUTF8)
	c.Response().WriteHeader(code)
	c.Response().Write(data)
	return
}

// Render HTML
//func (a *Application) Render(w io.Writer, name string, data interface{}) error {
//	return t.templates.ExecuteTemplate(w, name, data)
//}

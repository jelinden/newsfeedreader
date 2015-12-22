package main

import (
	"encoding/json"
	"fmt"
	"github.com/googollee/go-socket.io"
	"github.com/jelinden/newsfeedreader/service"
	"github.com/jelinden/newsfeedreader/util"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type (
	Template struct {
		templates *template.Template
	}
	Application struct {
		Sessions   *service.Sessions
		CookieUtil *util.CookieUtil
	}
)

func NewApplication() *Application {
	return &Application{}
}

func (a *Application) Init() {
	a.Sessions = service.NewSessions()
	a.CookieUtil = util.NewCookieUtil()
	a.Sessions.Init()
}

func (a *Application) Close() {
	a.Sessions.Close()
}

var newsFi, newsEn string

func (a *Application) tickNews(lang string) {
	for _ = range time.Tick(10 * time.Second) {
		rssList := a.Sessions.FetchRssItems(lang, 0, 5)
		if len(rssList) > 0 {
			result := map[string]interface{}{"news": rssList}
			news, err := json.Marshal(result)
			if err != nil {
				log.Println(err.Error())
			} else {
				if lang == "fi" {
					newsFi = string(news)
				} else {
					newsEn = string(news)
				}
			}
		} else {
			log.Println("Fetched rss list was empty")
		}
	}
}

func (a *Application) tickEmit(server *socketio.Server) {
	for _ = range time.Tick(10 * time.Second) {
		server.BroadcastTo("en", "message", newsEn)
		server.BroadcastTo("fi", "message", newsFi)
	}
}

func main() {
	app := NewApplication()
	app.Init()
	defer app.Close()
	go app.tickNews("fi")
	go app.tickNews("en")
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	go app.tickEmit(server)
	server.On("connection", func(so socketio.Socket) {
		pathArr := strings.Split(so.Request().Referer(), "/")
		path := pathArr[len(pathArr)-1]

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

	http.HandleFunc("/fi", func(w http.ResponseWriter, r *http.Request) {
		app.renderer("index_fi", "fi", w, r, t)
	})
	http.HandleFunc("/en", func(w http.ResponseWriter, r *http.Request) {
		app.renderer("index_en", "en", w, r, t)
	})
	http.Handle("/public/", http.StripPrefix("/public", http.FileServer(http.Dir("./public"))))
	http.Handle("/socket.io/", server)

	log.Fatal(http.ListenAndServe(":1300", nil))
}

func (a *Application) renderer(page string, lang string, w http.ResponseWriter, r *http.Request, t *Template) {
	t.Render(w, page, map[string]interface{}{"news": a.Sessions.FetchRssItems(lang, 0, 30)})
}

func (t *Template) Render(w io.Writer, name string, data interface{}) {
	t.templates.ExecuteTemplate(w, name, data)
}

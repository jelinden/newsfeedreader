package main

import (
	"html/template"
	"io"
	"log"
	"net/http"

	"encoding/json"
	"github.com/googollee/go-socket.io"
	"github.com/jelinden/newsfeedreader/service"
	"github.com/jelinden/newsfeedreader/util"
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

func main() {
	app := NewApplication()
	app.Init()
	defer app.Close()

	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.On("connection", func(so socketio.Socket) {
		so.Join("news")

		for _ = range time.Tick(10 * time.Second) {
			rssList := app.Sessions.FetchRssItems("fi")
			if len(rssList) > 0 {
				result := map[string]interface{}{"news": rssList}
				news, err := json.Marshal(result)
				if err != nil {
					log.Println(err.Error())
				} else {
					so.Emit("message", string(news))
					//so.BroadcastTo("news", "message", string(news))
				}
			} else {
				log.Println("Fetched rss list was empty")
			}
		}
		so.On("disconnection", func() {
			so.Leave("news")
		})
	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})
	/*
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			lang, err := r.Cookie("uutispuroLang")
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/en", 302)
				return
			} else if lang.Value == "fi" {
				http.Redirect(w, r, "/fi", 302)
				return
			} else {
				http.Redirect(w, r, "/en", 302)
				return
			}
		})
	*/
	t := &Template{
		templates: template.Must(template.ParseFiles("public/html/index_fi.html")),
	}

	http.HandleFunc("/fi", func(w http.ResponseWriter, r *http.Request) {
		app.renderer("index_fi", w, r, t)
	})
	http.Handle("/public/", http.StripPrefix("/public", http.FileServer(http.Dir("./public"))))
	http.Handle("/socket.io/", server)

	log.Fatal(http.ListenAndServe(":1300", nil))
}

func (a *Application) renderer(page string, w http.ResponseWriter, r *http.Request, t *Template) {
	t.Render(w, page, map[string]interface{}{"news": a.Sessions.FetchRssItems("fi")})
}

func (t *Template) Render(w io.Writer, name string, data interface{}) {
	t.templates.ExecuteTemplate(w, name, data)
}

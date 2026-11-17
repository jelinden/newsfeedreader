package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jelinden/newsfeedreader/app/middleware"
	"github.com/jelinden/newsfeedreader/app/render"
	"github.com/jelinden/newsfeedreader/app/routes"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/tick"
	"github.com/labstack/echo/v4"
	mw "github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/websocket"
	"golang.org/x/time/rate"
)

const (
	listenAddr      = ":1300"
	accessLogPath   = "access.log"
	staticCacheTTL  = 365 * 24 * 60 * 60 // seconds
	wsWriteDeadline = 5 * time.Second
	wsSendInterval  = 15 * time.Second
)

type Application struct {
	Mongo  *service.Mongo
	Tick   *tick.Tick
	Render *render.Render
}

func main() {
	app := &Application{}
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	e := echo.New()
	e.HideBanner = true
	e.Use(mw.RemoveTrailingSlashWithConfig(mw.TrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}))
	e.Use(mw.Recover())
	e.Use(mw.Secure())

	accessLog, err := setupMiddleware(e)
	if err != nil {
		log.Fatal(err)
	}
	defer accessLog.Close()

	go app.Tick.TickNews("fi")
	go app.Tick.TickNews("en")

	setupRoutes(e, app)

	log.Fatal(e.Start(listenAddr))
}

func (a *Application) Start() error {
	mongoURL := os.Getenv("MONGO_URL")
	if mongoURL == "" {
		return errors.New("MONGO_URL environment variable is not set")
	}
	mongo, err := service.NewMongo(mongoURL)
	if err != nil {
		return err
	}
	a.Mongo = mongo
	a.Tick = tick.NewTick(a.Mongo)

	render, err := render.NewRender(a.Mongo)
	if err != nil {
		a.Mongo.Close()
		return err
	}
	a.Render = render
	return nil
}

func (a *Application) Close() {
	log.Println("closing up")
	a.Mongo.Close()
}

func setupMiddleware(e *echo.Echo) (*middleware.LogCloser, error) {
	logger, accessLog, err := middleware.Logger(accessLogPath)
	if err != nil {
		return nil, err
	}
	e.Use(logger)
	return accessLog, nil
}

func setupRoutes(e *echo.Echo, app *Application) {
	// NOTE: the group prefix must be "" rather than "/": echo concatenates the
	// prefix with the route path, so "/" + "/fi" would register "//fi".
	paths := e.Group("")
	paths.Use(mw.Gzip())

	for _, lang := range []string{"fi", "en"} {
		paths.GET("/"+lang, routes.Index(app.Render, lang))
		paths.GET("/"+lang+"/:page", routes.Index(app.Render, lang))
		paths.GET("/"+lang+"/login", routes.Login(app.Render, lang))
		paths.GET("/"+lang+"/search", routes.Search(app.Render, lang))
		paths.GET("/"+lang+"/search/:page", routes.Search(app.Render, lang))
		paths.GET("/"+lang+"/category/:category", redirect)
		paths.GET("/"+lang+"/category/:category/:page", routes.Category(app.Render, lang))
		paths.GET("/"+lang+"/source/:source", redirect)
		paths.GET("/"+lang+"/source/:source/:page", routes.Source(app.Render, lang))
	}
	paths.GET("/", routes.Root)
	paths.GET("/uutiset/fi", redirectRoot)
	paths.GET("/uutiset/en", redirectRoot)

	// Click tracking and the json api are write/DB heavy endpoints; limit them
	// per IP so they cannot be used to hammer the database.
	api := e.Group("/api", rateLimiter(2, 20))
	api.GET("/click/:id", routes.Click(app.Mongo))
	api.GET("/news", routes.News(app.Mongo))

	paths.GET("/public/:filePath/:fileName", static)
	paths.File("/favicon.ico", "public/img/favicon.ico")
	paths.File("/sitemap.xml", "public/sitemap.xml")
	paths.File("/robots.txt", "public/robots.txt")
	paths.File("/ads.txt", "public/ads.txt")
	paths.GET("/serviceworker.js", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/javascript")
		return c.File("public/js/serviceworker.js")
	})

	wss := e.Group("/ws", rateLimiter(1, 10))
	wss.GET("/:channel", ws(app.Tick))
}

// rateLimiter limits requests per IP using a token bucket. rate is the
// sustained requests per second; burst absorbs legitimate spikes such as a
// page firing several click beacons or a user opening multiple tabs.
func rateLimiter(ratePerSecond rate.Limit, burst int) echo.MiddlewareFunc {
	return mw.RateLimiterWithConfig(mw.RateLimiterConfig{
		Store: mw.NewRateLimiterMemoryStoreWithConfig(mw.RateLimiterMemoryStoreConfig{
			Rate:      ratePerSecond,
			Burst:     burst,
			ExpiresIn: 10 * time.Minute,
		}),
	})
}

func redirect(c echo.Context) error {
	c.Response().Header().Set("Location", c.Request().URL.Path+"/0")
	return c.NoContent(http.StatusMovedPermanently)
}

func redirectRoot(c echo.Context) error {
	lang := "fi"
	if strings.HasSuffix(c.Request().URL.Path, "en") {
		lang = "en"
	}
	c.Response().Header().Set("Location", "/"+lang+"/0")
	return c.NoContent(http.StatusMovedPermanently)
}

var allowedStaticDirs = map[string]bool{"js": true, "img": true, "css": true}

func static(c echo.Context) error {
	dir := c.Param("filePath")
	if !allowedStaticDirs[dir] {
		return c.NoContent(http.StatusNotFound)
	}
	// Defense in depth: even though the router normalizes paths, make sure
	// the resolved file stays inside the allowed directory.
	name := filepath.Base(c.Param("fileName"))
	if name != c.Param("fileName") || !strings.Contains(name, ".") || strings.HasPrefix(name, ".") {
		return c.NoContent(http.StatusNotFound)
	}
	c.Response().Header().Set("Cache-Control", "max-age="+strconv.Itoa(staticCacheTTL)+", public, must-revalidate, proxy-revalidate")
	return c.File(filepath.Join("public", dir, name))
}

func ws(tick *tick.Tick) echo.HandlerFunc {
	return func(c echo.Context) error {
		channel := c.Param("channel")
		if channel != "fi" && channel != "en" {
			return c.NoContent(http.StatusNotFound)
		}
		// websocket.Handler rejects clients that send no Origin header, which
		// breaks non-browser clients. The feed is public and read-only, so any
		// origin is accepted; the origin is still recorded on the connection.
		server := &websocket.Server{
			Handshake: func(config *websocket.Config, req *http.Request) error {
				origin, err := websocket.Origin(config, req)
				config.Origin = origin
				return err
			},
			Handler: func(ws *websocket.Conn) {
				defer ws.Close()
				ws.PayloadType = websocket.TextFrame
				ticker := time.NewTicker(wsSendInterval)
				defer ticker.Stop()
				// Send the current state immediately, then on every tick.
				for {
					ws.SetWriteDeadline(time.Now().Add(wsWriteDeadline))
					if err := websocket.Message.Send(ws, tick.Latest(channel)); err != nil {
						return
					}
					if _, ok := <-ticker.C; !ok {
						return
					}
				}
			},
		}
		server.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

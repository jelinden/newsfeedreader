package render

import (
	"bytes"
	"html/template"
	"log"
	"strings"

	"github.com/jelinden/newsfeedreader/app/domain"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/util"
	"github.com/labstack/echo/v4"
	"github.com/rsniezynski/go-asset-helper"
)

type (
	Render struct {
		Mongo  *service.Mongo
		t      *Template
		static *asset.Static
	}
	Template struct {
		templates *template.Template
	}
)

const pageSize = 30

func NewRender(mongo *service.Mongo) (*Render, error) {
	newStatic, err := asset.NewStatic("", "./manifest.json")
	if err != nil {
		return nil, err
	}
	funcs := newStatic.FuncMap()
	funcs["minus"] = func(a, b int) int { return a - b }
	funcs["add"] = func(a, b int) int { return a + b }
	funcs["toLower"] = strings.ToLower

	templates, err := template.New("").Funcs(funcs).ParseGlob("public/html/*")
	if err != nil {
		return nil, err
	}
	return &Render{
		Mongo:  mongo,
		t:      &Template{templates: templates},
		static: newStatic,
	}, nil
}

func (r *Render) Index(name string, lang string, page int, c echo.Context, statusCode int) error {
	rssList := r.Mongo.FetchRssItems(lang, page, pageSize)
	news := r.baseNews(lang, page)
	news.ResultCount = len(rssList)
	news.RSS = rssList
	return r.renderPage(name, news, c, statusCode)
}

func (r *Render) Login(name string, lang string, c echo.Context, statusCode int) error {
	return r.renderPage(name, r.baseNews(lang, 0), c, statusCode)
}

func (r *Render) RenderSearch(name string, lang string, searchString string, page int, c echo.Context, statusCode int) error {
	rssList := r.Mongo.Search(searchString, lang, page, pageSize)
	news := r.baseNews(lang, page)
	news.ResultCount = len(rssList)
	news.SearchQuery = searchString
	news.RSS = rssList
	return r.renderPage(name, news, c, statusCode)
}

func (r *Render) ByCategory(name string, lang string, category string, page int, c echo.Context, statusCode int) error {
	rssList := r.Mongo.FetchRssItemsByCategory(lang, category, page, pageSize)
	news := r.baseNews(lang, page)
	news.ResultCount = len(rssList)
	news.Category = category
	if lang == "en" {
		news.CategoryEnName = util.EnCategoryName(category)
	}
	news.RSS = rssList
	return r.renderPage(name, news, c, statusCode)
}

func (r *Render) BySource(name string, lang string, source string, page int, c echo.Context, statusCode int) error {
	rssList := r.Mongo.FetchRssItemsBySource(lang, source, page, pageSize)
	news := r.baseNews(lang, page)
	news.ResultCount = len(rssList)
	news.Source = source
	news.RSS = rssList
	return r.renderPage(name, news, c, statusCode)
}

// baseNews fills the parts of the payload every page shares.
func (r *Render) baseNews(lang string, page int) *domain.News {
	return &domain.News{
		Page:         page,
		Lang:         lang,
		MostReadList: r.Mongo.MostReadWeekly(lang, 5),
	}
}

func (r *Render) renderPage(name string, news *domain.News, c echo.Context, statusCode int) error {
	var buf bytes.Buffer
	if err := r.t.templates.ExecuteTemplate(&buf, name, news); err != nil {
		log.Println("rendering page", name, "failed.", err.Error())
		return err
	}
	return r.render(statusCode, buf.Bytes(), c)
}

func (r *Render) render(code int, data []byte, c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(code)
	_, err := c.Response().Write(data)
	return err
}

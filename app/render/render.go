package render

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strconv"
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

func NewRender(mongo *service.Mongo) *Render {
	render := &Render{}
	render.Mongo = mongo
	newStatic, _ := asset.NewStatic("", "./manifest.json")
	render.static = newStatic
	render.t = &Template{
		templates: template.Must(template.New("").Funcs(newStatic.FuncMap()).Funcs(template.FuncMap{
			"minus": func(a, b int) int {
				return a - b
			},
			"add": func(a, b int) int {
				return a + b
			},
			"toLower": strings.ToLower,
		}).ParseGlob("public/html/*")),
	}
	return render
}

func (r *Render) Index(name string, lang string, page int, c echo.Context, statusCode int) error {
	return r.render(http.StatusOK, r.RenderIndex(name, lang, strconv.Itoa(page)), c)
}

func (r *Render) RenderIndex(params ...string) []byte {
	p, _ := strconv.Atoi(params[2])
	buf := r.getIndexTemplate(params[0], params[1], p)
	return buf.Bytes()
}

func (r *Render) getIndexTemplate(name string, lang string, page int) bytes.Buffer {
	var buf bytes.Buffer
	rssList := r.Mongo.FetchRssItems(lang, page, 30)
	mostReadList := r.Mongo.MostReadWeekly(lang, 0, 5)
	err := r.t.templates.ExecuteTemplate(&buf, name, &domain.News{
		Page:         page,
		Lang:         lang,
		ResultCount:  len(rssList),
		RSS:          rssList,
		MostReadList: mostReadList,
	})
	if err != nil {
		log.Println("rendering page", name, "failed.", err.Error())
	}
	return buf
}

func (r *Render) Login(name string, lang string, c echo.Context, statusCode int) error {
	buf := r.RenderLogin(name, lang)
	return r.render(http.StatusOK, buf.Bytes(), c)
}

func (r *Render) RenderLogin(name string, lang string) bytes.Buffer {
	return r.getLoginTemplate(name, lang)
}

func (r *Render) getLoginTemplate(name string, lang string) bytes.Buffer {
	var buf bytes.Buffer
	mostReadList := r.Mongo.MostReadWeekly(lang, 0, 5)
	err := r.t.templates.ExecuteTemplate(&buf, name, &domain.News{
		Lang:         lang,
		MostReadList: mostReadList,
	})
	if err != nil {
		log.Println("rendering page", name, "failed.", err.Error())
	}
	return buf
}

func (r *Render) RenderSearch(name string, lang string, searchString string, page int, c echo.Context, statusCode int) error {
	var buf bytes.Buffer
	rssList := r.Mongo.Search(searchString, lang, page, 30)
	mostReadList := r.Mongo.MostReadWeekly(lang, 0, 5)
	err := r.t.templates.ExecuteTemplate(&buf, name, &domain.News{
		Page:         page,
		Lang:         lang,
		ResultCount:  len(rssList),
		SearchQuery:  searchString,
		RSS:          rssList,
		MostReadList: mostReadList,
	})
	if err != nil {
		log.Println("rendering page", name, "failed.", err.Error())
		return err
	}
	return r.render(http.StatusOK, buf.Bytes(), c)
}

func (r *Render) ByCategory(name string, lang string, category string, page int, c echo.Context, statusCode int) error {
	return r.render(http.StatusOK, r.RenderByCategory(name, lang, category, strconv.Itoa(page)), c)
}

func (r *Render) RenderByCategory(params ...string) []byte {
	p, _ := strconv.Atoi(params[3])
	return r.getCategoryTemplate(params[0], params[1], params[2], p).Bytes()
}

func (r *Render) getCategoryTemplate(name string, lang string, category string, page int) *bytes.Buffer {
	var buf bytes.Buffer
	rssList := r.Mongo.FetchRssItemsByCategory(lang, category, page, 30)
	mostReadList := r.Mongo.MostReadWeekly(lang, 0, 5)
	var catEn string
	if lang == "en" {
		catEn = util.EnCategoryName(category)
	}
	err := r.t.templates.ExecuteTemplate(&buf, name, &domain.News{
		Page:           page,
		Lang:           lang,
		ResultCount:    len(rssList),
		Category:       category,
		CategoryEnName: catEn,
		RSS:            rssList,
		MostReadList:   mostReadList,
	})
	if err != nil {
		log.Println("rendering page", name, "failed.", err.Error())
	}
	return &buf
}

func (r *Render) BySource(name string, lang string, source string, page int, c echo.Context, statusCode int) error {
	return r.render(http.StatusOK, r.RenderBySource(name, lang, source, strconv.Itoa(page)), c)

}

func (r *Render) RenderBySource(params ...string) []byte {
	p, _ := strconv.Atoi(params[3])
	return r.getSourceTemplate(params[0], params[1], params[2], p).Bytes()
}

func (r *Render) getSourceTemplate(name string, lang string, source string, page int) *bytes.Buffer {
	var buf bytes.Buffer
	rssList := r.Mongo.FetchRssItemsBySource(lang, source, page, 30)
	mostReadList := r.Mongo.MostReadWeekly(lang, 0, 5)
	err := r.t.templates.ExecuteTemplate(&buf, name, &domain.News{
		Page:         page,
		Lang:         lang,
		ResultCount:  len(rssList),
		Source:       source,
		RSS:          rssList,
		MostReadList: mostReadList,
	})
	if err != nil {
		log.Println("rendering page", name, "failed.", err.Error())
	}
	return &buf
}

func (r *Render) render(code int, data []byte, c echo.Context) (err error) {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(code)
	c.Response().Write(data)
	return
}

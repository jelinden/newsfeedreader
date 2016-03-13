package render

import (
	"bytes"
	"github.com/jelinden/newsfeedreader/app/domain"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/util"
	"github.com/labstack/echo"
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
	Render struct {
		Mongo          *service.Mongo
		t              *Template
		cache          *ttlcache.Cache
		static         *asset.Static
		newsFi, newsEn string
	}
	Template struct {
		templates *template.Template
	}
)

func NewRender(mongo *service.Mongo) *Render {
	render := &Render{}
	render.Mongo = mongo
	render.cache = ttlcache.NewCache(time.Minute)

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

func (r *Render) RenderIndex(name string, lang string, page int, c echo.Context, statusCode int) error {
	pString := strconv.Itoa(page)
	key := name + "_" + pString
	value, exists := r.cache.Get(key)
	if exists {
		log.Println("found from cache", r.cache.Count())
		return r.render(http.StatusOK, name, []byte(value), c)
	} else {
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
			return err
		}
		r.cache.Set(key, buf.String())
		log.Println("cache count after add", r.cache.Count())
		return r.render(http.StatusOK, name, buf.Bytes(), c)
	}
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
	return r.render(http.StatusOK, name, buf.Bytes(), c)
}

func (r *Render) RenderByCategory(name string, lang string, category string, page int, c echo.Context, statusCode int) error {
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
		return err
	}
	return r.render(http.StatusOK, name, buf.Bytes(), c)
}

func (r *Render) RenderBySource(name string, lang string, source string, page int, c echo.Context, statusCode int) error {
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
		return err
	}
	return r.render(http.StatusOK, name, buf.Bytes(), c)
}

func (r *Render) render(code int, name string, data []byte, c echo.Context) (err error) {
	c.Response().Header().Set(echo.ContentType, echo.TextHTMLCharsetUTF8)
	c.Response().WriteHeader(code)
	c.Response().Write(data)
	return
}

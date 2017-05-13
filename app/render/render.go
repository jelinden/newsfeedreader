package render

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jelinden/newsfeedreader/app/domain"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/util"
	"github.com/labstack/echo"
	"github.com/rsniezynski/go-asset-helper"
)

type (
	Render struct {
		Mongo          *service.Mongo
		t              *Template
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
	go doEvery(30*time.Second, render.updateStaleCacheItems)
	go render.warmUpCache()
	return render
}

func (r *Render) warmUpCache() {
	keys := []string{"category_fi|fi|Urheilu|0",
		"category_fi|fi|Matkustus|0",
		"category_fi|fi|Ulkomaat|0",
		"category_fi|fi|Tiede|0",
		"category_fi|fi|Talous|0",
		"category_fi|fi|Kulttuuri|0",
		"category_fi|fi|Pelit|0",
		"category_fi|fi|Viihde|0",
		"category_fi|fi|Blogs|0",
		"category_fi|fi|Kotimaa|0",
		"category_fi|fi|Digi|0",
		"category_fi|fi|Asuminen|0",
		"category_fi|fi|Ruoka|0",
		"category_fi|fi|Terveys|0",
		"category_fi|fi|Elokuvat|0",
		"category_fi|fi|Naisetjamuoti|0",
		"category_en|en|Urheilu|0",
		"category_en|en|Matkustus|0",
		"category_en|en|Ulkomaat|0",
		"category_en|en|Tiede|0",
		"category_en|en|Talous|0",
		"category_en|en|Kulttuuri|0",
		"category_en|en|Pelit|0",
		"category_en|en|Viihde|0",
		"category_en|en|Blogs|0",
		"category_en|en|Kotimaa|0",
		"category_en|en|Digi|0",
		"category_en|en|Asuminen|0",
		"category_en|en|Ruoka|0",
		"category_en|en|Terveys|0",
		"category_en|en|Elokuvat|0",
		"category_en|en|Naisetjamuoti|0",
	}
	r.RenderIndex("index_fi|fi|0")
	r.RenderIndex("index_en|en|0")
	for _, key := range keys {
		r.RenderByCategory(key)
	}
}

func (r *Render) Index(name string, lang string, page int, c echo.Context, statusCode int) error {
	pString := strconv.Itoa(page)
	key := name + "|" + lang + "|" + pString
	cacheItem := util.GetItemFromCache(key)
	if cacheItem != nil {
		return r.render(http.StatusOK, name, cacheItem.Value, c)
	}
	buf := r.RenderIndex(key)
	return r.render(http.StatusOK, name, buf.Bytes(), c)
}

func (r *Render) RenderIndex(key string) bytes.Buffer {
	splittedKey := strings.Split(key, "|")
	lang, _ := strconv.Atoi(splittedKey[2])
	buf := r.getIndexTemplate(splittedKey[0], splittedKey[1], lang)
	util.AddItemToCache(key, "index", buf.Bytes(), 30*time.Second)
	return buf
}

func (r *Render) Login(name string, lang string, c echo.Context, statusCode int) error {
	key := name + "|" + lang
	cacheItem := util.GetItemFromCache(key)
	if cacheItem != nil {
		return r.render(http.StatusOK, name, cacheItem.Value, c)
	}
	buf := r.RenderLogin(key)
	return r.render(http.StatusOK, name, buf.Bytes(), c)
}

func (r *Render) RenderLogin(key string) bytes.Buffer {
	splittedKey := strings.Split(key, "|")
	buf := r.getLoginTemplate(splittedKey[0], splittedKey[1])
	util.AddItemToCache(key, "login", buf.Bytes(), 30*time.Second)
	return buf
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
	return r.render(http.StatusOK, name, buf.Bytes(), c)
}

func (r *Render) ByCategory(name string, lang string, category string, page int, c echo.Context, statusCode int) error {
	pString := strconv.Itoa(page)
	key := name + "|" + lang + "|" + category + "|" + pString
	cacheItem := util.GetItemFromCache(key)
	if cacheItem != nil {
		return r.render(http.StatusOK, name, cacheItem.Value, c)
	}
	buf := r.RenderByCategory(key)
	return r.render(http.StatusOK, name, buf.Bytes(), c)
}

func (r *Render) RenderByCategory(key string) bytes.Buffer {
	splittedKey := strings.Split(key, "|")
	lang, _ := strconv.Atoi(splittedKey[3])
	buf := r.getCategoryTemplate(splittedKey[0], splittedKey[1], splittedKey[2], lang)
	util.AddItemToCache(key, "category", buf.Bytes(), 30*time.Second)
	return *buf
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
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(code)
	c.Response().Write(data)
	return
}

func (r *Render) updateStaleCacheItems() {
	for _, item := range util.Cache.Items() {
		cItem := item.(util.CacheItem)
		if time.Now().After(cItem.Expire) {
			if cItem.Type == "index" {
				r.RenderIndex(cItem.Key)
			} else if cItem.Type == "login" {
				r.RenderLogin(cItem.Key)
			} else if cItem.Type == "category" {
				r.RenderByCategory(cItem.Key)
			}
		}
	}
}

func doEvery(d time.Duration, f func()) {
	for range time.Tick(d) {
		f()
	}
}

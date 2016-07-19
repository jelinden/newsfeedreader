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
	"github.com/muesli/cache2go"
	"github.com/rsniezynski/go-asset-helper"
)

type (
	Render struct {
		Mongo          *service.Mongo
		t              *Template
		static         *asset.Static
		newsFi, newsEn string
		expiringCache  *cache2go.CacheTable
	}
	Template struct {
		templates *template.Template
	}
)

func NewRender(mongo *service.Mongo) *Render {
	render := &Render{}
	render.Mongo = mongo
	render.expiringCache = cache2go.Cache("news")
	render.expiringCache.SetAddedItemCallback(func(entry *cache2go.CacheItem) {
		//log.Println("Added:", entry.Key(), entry.CreatedOn())
	})
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
	//log.Println("rendering index", name)
	res, err := r.expiringCache.Value(key)
	if err == nil {
		//og.Println("Found value in cache")
		return r.render(http.StatusOK, name, res.Data().([]byte), c)
	}
	buf := r.getIndexTemplate(name, lang, page)
	item := r.expiringCache.Add(key, 30*time.Second, buf.Bytes())
	go r.expireIndexCallback(item, name, lang, page)
	return r.render(http.StatusOK, name, buf.Bytes(), c)
}

func (r *Render) expireIndexCallback(item *cache2go.CacheItem, name string, lang string, page int) {
	item.SetAboutToExpireCallback(func(key interface{}) {
		go r.addItemTocache(key.(string), name, lang, page, r.getIndexTemplate(name, lang, page))
	})
}

func (r *Render) addItemTocache(key string, name string, lang string, page int, tmpl bytes.Buffer) {
	for _, err := r.expiringCache.Value(key); err == nil; {
		time.Sleep(1 * time.Millisecond)
	}
	cachedItem := r.expiringCache.Add(key, 30*time.Second, tmpl.Bytes())
	r.expireIndexCallback(cachedItem, name, lang, page)
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
	pString := strconv.Itoa(page)
	key := name + "_" + category + "_" + pString
	//log.Println("rendering byCategory", key)
	res, err := r.expiringCache.Value(key)
	if err == nil {
		//log.Println("Found value in cache")
		return r.render(http.StatusOK, name, res.Data().([]byte), c)
	}
	buf := r.getCategoryTemplate(name, lang, category, page)
	item := r.expiringCache.Add(key, 30*time.Second, buf.Bytes())
	go r.expireCategoryCallback(item, name, lang, category, page)
	return r.render(http.StatusOK, name, buf.Bytes(), c)
}

func (r *Render) expireCategoryCallback(item *cache2go.CacheItem, name string, lang string, category string, page int) {
	item.SetAboutToExpireCallback(func(key interface{}) {
		go r.addCategoryItemTocache(key.(string), name, lang, category, page, *r.getCategoryTemplate(name, lang, category, page))
	})
}

func (r *Render) addCategoryItemTocache(key string, name string, lang string, category string, page int, tmpl bytes.Buffer) {
	for _, err := r.expiringCache.Value(key); err == nil; {
		time.Sleep(1 * time.Millisecond)
	}
	cachedItem := r.expiringCache.Add(key, 30*time.Second, tmpl.Bytes())
	r.expireCategoryCallback(cachedItem, name, lang, category, page)
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

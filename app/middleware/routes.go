package middleware

import (
	"github.com/jelinden/newsfeedreader/app/render"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/util"
	"github.com/labstack/echo"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func Root() echo.HandlerFunc {
	return func(c echo.Context) error {
		lang := c.Request().Header().Get("Accept-Language")
		if lang == "" {
			return c.Redirect(http.StatusFound, "/fi")
		} else if strings.Contains(strings.Split(lang, ",")[0], "en") {
			return c.Redirect(http.StatusFound, "/en")
		}
		return c.Redirect(http.StatusFound, "/fi")
	}
}

func FiRoot(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		return render.RenderIndex("index_fi", "fi", 0, c, http.StatusOK)
	}
}

func EnRoot(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		return render.RenderIndex("index_en", "en", 0, c, http.StatusOK)
	}
}

func FiRootPaged(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return render.RenderIndex("index_fi", "fi", page, c, http.StatusOK)
			}
		}
		return render.RenderIndex("index_fi", "fi", 0, c, http.StatusBadRequest)
	}
}

func EnRootPaged(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return render.RenderIndex("index_en", "en", page, c, http.StatusOK)
			}
		}
		return render.RenderIndex("index_en", "en", 0, c, http.StatusBadRequest)
	}
}

func FiSearch(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		return render.RenderSearch("search_fi", "fi", validateAndCorrectifySearchTerm(c.FormValue("q")), 0, c, http.StatusOK)
	}
}
func EnSearch(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		return render.RenderSearch("search_en", "en", validateAndCorrectifySearchTerm(c.FormValue("q")), 0, c, http.StatusOK)
	}
}
func FiSearchPaged(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return render.RenderSearch("search_fi", "fi", validateAndCorrectifySearchTerm(c.FormValue("q")), page, c, http.StatusOK)
			}
		}
		return render.RenderSearch("search_fi", "fi", validateAndCorrectifySearchTerm(c.FormValue("q")), 0, c, http.StatusOK)
	}
}
func EnSearchPaged(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		if page, err := strconv.Atoi(c.P(0)); err == nil {
			if page < 999 && page >= 0 {
				return render.RenderSearch("search_en", "en", validateAndCorrectifySearchTerm(c.FormValue("q")), page, c, http.StatusOK)
			}
		}
		return render.RenderSearch("search_en", "en", validateAndCorrectifySearchTerm(c.FormValue("q")), 0, c, http.StatusOK)
	}
}
func FiCategory(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		category := util.ToUpper(c.P(0))
		if page, err := strconv.Atoi(c.P(1)); err == nil {
			if page < 999 && page >= 0 {
				return render.RenderByCategory("category_fi", "fi", validateAndCorrectifySearchTerm(category), page, c, http.StatusOK)
			}
		}
		return render.RenderByCategory("category_fi", "fi", validateAndCorrectifySearchTerm(category), 0, c, http.StatusOK)
	}
}
func EnCategory(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		category := util.ToUpper(c.P(0))
		if page, err := strconv.Atoi(c.P(1)); err == nil {
			if page < 999 && page >= 0 {
				return render.RenderByCategory("category_en", "en", validateAndCorrectifySearchTerm(category), page, c, http.StatusOK)
			}
		}
		return render.RenderByCategory("category_en", "en", validateAndCorrectifySearchTerm(category), 0, c, http.StatusOK)
	}
}
func FiSource(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		category := util.ToUpper(c.P(0))
		if page, err := strconv.Atoi(c.P(1)); err == nil {
			if page < 999 && page >= 0 {
				return render.RenderBySource("source_fi", "fi", validateAndCorrectifySearchTerm(category), page, c, http.StatusOK)
			}
		}
		return render.RenderBySource("source_fi", "fi", validateAndCorrectifySearchTerm(category), 0, c, http.StatusOK)
	}
}
func EnSource(render *render.Render) echo.HandlerFunc {
	return func(c echo.Context) error {
		category := util.ToUpper(c.P(0))
		if page, err := strconv.Atoi(c.P(1)); err == nil {
			if page < 999 && page >= 0 {
				return render.RenderBySource("source_en", "en", validateAndCorrectifySearchTerm(category), page, c, http.StatusOK)
			}
		}
		return render.RenderBySource("source_en", "en", validateAndCorrectifySearchTerm(category), 0, c, http.StatusOK)
	}
}
func Click(mgo *service.Mongo) echo.HandlerFunc {
	return func(c echo.Context) error {
		mgo.SaveClick(validateAndCorrectifySearchTerm(c.P(0)))
		return c.NoContent(http.StatusOK)
	}
}

func validateAndCorrectifySearchTerm(searchString string) string {
	r, _ := regexp.Compile("[^-a-zåäöA-ZÅÄÖ0-9 ]+")
	return string(r.ReplaceAll([]byte(searchString), []byte(""))[:])
}

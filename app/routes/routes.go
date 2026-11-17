package routes

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jelinden/newsfeedreader/app/render"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/util"

	"github.com/labstack/echo/v4"
)

const maxPages = 999

// searchTermPattern strips anything that is not a letter (incl. Finnish
// letters), a digit, a space or a hyphen from user-supplied search terms.
// Compiling it once avoids recompiling on every request.
var searchTermPattern = regexp.MustCompile(`[^-a-zåäöA-ZÅÄÖ0-9 ]+`)

// ValidateAndCorrectifySearchTerm sanitizes user input used in mongo queries
// and templates.
func ValidateAndCorrectifySearchTerm(searchString string) string {
	return searchTermPattern.ReplaceAllString(searchString, "")
}

// parsePage returns the page number from a path parameter. An absent
// parameter means the first page (e.g. /fi is /fi/0); invalid, negative or
// absurdly large pages resolve to page 0 with http.StatusBadRequest.
func parsePage(raw string) (int, int) {
	if raw == "" {
		return 0, http.StatusOK
	}
	if page, err := strconv.Atoi(raw); err == nil && page >= 0 && page < maxPages {
		return page, http.StatusOK
	}
	return 0, http.StatusBadRequest
}

func Root(c echo.Context) error {
	lang := "fi"
	if strings.Contains(c.Request().Header.Get("Accept-Language"), "en") {
		lang = "en"
	}
	return c.Redirect(http.StatusFound, "/"+lang)
}

// Index serves the front page of a language, e.g. /fi and /fi/0.
func Index(render *render.Render, lang string) echo.HandlerFunc {
	return func(c echo.Context) error {
		page, status := parsePage(c.Param("page"))
		return render.Index("index_"+lang, lang, page, c, status)
	}
}

// Search serves search results for a language, e.g. /en/search?q=news.
func Search(render *render.Render, lang string) echo.HandlerFunc {
	return func(c echo.Context) error {
		page, status := parsePage(c.Param("page"))
		query := ValidateAndCorrectifySearchTerm(c.FormValue("q"))
		return render.RenderSearch("search_"+lang, lang, query, page, c, status)
	}
}

// Category serves news of a single category, e.g. /fi/category/urheilu/0.
func Category(render *render.Render, lang string) echo.HandlerFunc {
	return func(c echo.Context) error {
		page, status := parsePage(c.Param("page"))
		category := util.ToUpper(ValidateAndCorrectifySearchTerm(c.Param("category")))
		return render.ByCategory("category_"+lang, lang, category, page, c, status)
	}
}

// Source serves news of a single rss source, e.g. /fi/source/yle/0.
func Source(render *render.Render, lang string) echo.HandlerFunc {
	return func(c echo.Context) error {
		page, status := parsePage(c.Param("page"))
		source := util.ToUpper(ValidateAndCorrectifySearchTerm(c.Param("source")))
		return render.BySource("source_"+lang, lang, source, page, c, status)
	}
}

func Login(render *render.Render, lang string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return render.Login("login", lang, c, http.StatusOK)
	}
}

// Click registers a click on a news item.
func Click(mongo *service.Mongo) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if !service.IsValidID(id) {
			return c.NoContent(http.StatusBadRequest)
		}
		if err := mongo.SaveClick(id); err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.NoContent(http.StatusOK)
	}
}

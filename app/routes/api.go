package routes

import (
	"net/http"
	"strings"

	"github.com/jelinden/newsfeedreader/app/domain"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/labstack/echo/v4"
)

type NewsItems struct {
	Items []domain.RSS `json:"items"`
}

// News returns news items as json. Example: /api/news?q=traffic,climate
func News(mongo *service.Mongo) echo.HandlerFunc {
	return func(c echo.Context) error {
		searchString := BuildSearchQuery(c.QueryParam("q"))
		news := mongo.Search(searchString, "en", 0, 20)
		return c.JSON(http.StatusOK, NewsItems{Items: news})
	}
}

// BuildSearchQuery turns a comma separated list of terms into a mongo $text
// search string that matches every term as an exact phrase. Mongo combines
// $text terms with AND, e.g. `traffic,climate` -> `"traffic" "climate"`.
// Each term is sanitized with ValidateAndCorrectifySearchTerm first.
func BuildSearchQuery(params string) string {
	terms := make([]string, 0, 4)
	for _, item := range strings.Split(params, ",") {
		if term := ValidateAndCorrectifySearchTerm(item); term != "" {
			terms = append(terms, `"`+term+`"`)
		}
	}
	return strings.Join(terms, " ")
}

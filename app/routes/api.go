package routes

import (
	"net/http"
	"strings"

	"github.com/jelinden/newsfeedreader/app/domain"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/labstack/echo/v4"
)

// News returns news items as json
func News(c echo.Context) error {
	params := c.QueryParam("q")
	paramSlice := validateParams(strings.Split(params, ","))
	p := `"` + strings.Join(paramSlice, `" "`) + `"`
	news := NewsItems{Items: service.News(p)}
	return c.JSON(http.StatusOK, news)
}

func validateParams(params []string) []string {
	for i, item := range params {
		params[i] = validateAndCorrectifySearchTerm(item)
	}
	return params
}

type NewsItems struct {
	Items []domain.RSS `json:"items"`
}

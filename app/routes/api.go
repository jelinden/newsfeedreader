package routes

import (
	"net/http"
	"strings"

	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/labstack/echo"
)

// News returns news items as json
func News(c echo.Context) error {
	params := c.QueryParam("q")
	paramSlice := validateParams(strings.Split(params, ","))
	p := `"` + strings.Join(paramSlice, `" "`) + `"`
	news := service.News(p)
	return c.JSON(http.StatusOK, news)
}

func validateParams(params []string) []string {
	for i, item := range params {
		params[i] = validateAndCorrectifySearchTerm(item)
	}
	return params
}

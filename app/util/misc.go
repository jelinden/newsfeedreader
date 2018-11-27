package util

import (
	"time"
	"unicode"

	"github.com/jelinden/newsfeedreader/app/domain"
)

func DoEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}

func AddCategoryEnNames(items []domain.RSS) []domain.RSS {
	for i := range items {
		cat := items[i].Category.CategoryName
		items[i].Category.CategoryEnName = EnCategoryName(cat)
	}
	return items
}

func EnCategoryName(cat string) string {
	if cat == "Digi" {
		return "Digital media"
	} else if cat == "Elokuvat" {
		return "TV and movies"
	} else if cat == "Koti" || cat == "Asuminen" {
		return "Home and living"
	} else if cat == "Kotimaa" {
		return "Domestic"
	} else if cat == "Kulttuuri" {
		return "Culture"
	} else if cat == "Matkustus" {
		return "Travel"
	} else if cat == "Pelit" {
		return "Games"
	} else if cat == "Ruoka" {
		return "Food"
	} else if cat == "Talous" {
		return "Economy"
	} else if cat == "Terveys" {
		return "Health"
	} else if cat == "Tiede" {
		return "Science"
	} else if cat == "Ulkomaat" {
		return "Foreign"
	} else if cat == "Urheilu" {
		return "Sports"
	} else if cat == "Viihde" {
		return "Entertainment"
	} else if cat == "Blogs" {
		return "Blogs"
	} else if cat == "Naisetjamuoti" {
		return "Women and fashion"
	} else {
		return ""
	}
}

func ToUpper(textString string) string {
	if textString != "" {
		text := []rune(textString)
		text[0] = unicode.ToUpper(text[0])
		return string(text)
	}
	return textString
}

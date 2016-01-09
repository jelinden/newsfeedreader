package util

import (
	"github.com/jelinden/newsfeedreader/app/domain"
	"time"
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
		return "Tech"
	} else if cat == "Elokuvat" {
		return "Movies"
	} else if cat == "Koti" {
		return "Home"
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
	} else if cat == "Blogit" {
		return "Blogs"
	} else if cat == "Naiset" {
		return "Women"
	} else {
		return ""
	}
}

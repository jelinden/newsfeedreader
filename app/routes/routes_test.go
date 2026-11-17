package routes

import (
	"net/http"
	"testing"
)

func TestValidateAndCorrectifySearchTerm(t *testing.T) {
	cases := map[string]string{
		"hello world":             "hello world",
		"Mobiili-Blogs":           "Mobiili-Blogs",
		"äöåÄÖÅ 123":              "äöåÄÖÅ 123",
		`<script>x</scr` + "ipt>": "scriptxscript",
		"foo$bar":                 "foobar",
		"\"quoted\"":              "quoted",
		"":                        "",
	}
	for input, want := range cases {
		if got := ValidateAndCorrectifySearchTerm(input); got != want {
			t.Errorf("ValidateAndCorrectifySearchTerm(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildSearchQuery(t *testing.T) {
	cases := map[string]string{
		"traffic,climate":   `"traffic" "climate"`,
		"traffic":           `"traffic"`,
		"traffic,":          `"traffic"`,
		",,":                "",
		"":                  "",
		"a b,c":             `"a b" "c"`,
		`ev!il,"<img oner"`: `"evil" "img oner"`,
	}
	for input, want := range cases {
		if got := BuildSearchQuery(input); got != want {
			t.Errorf("BuildSearchQuery(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParsePage(t *testing.T) {
	cases := []struct {
		input      string
		wantPage   int
		wantStatus int
	}{
		{"0", 0, http.StatusOK},
		{"42", 42, http.StatusOK},
		{"998", 998, http.StatusOK},
		{"999", 0, http.StatusBadRequest},
		{"-1", 0, http.StatusBadRequest},
		{"abc", 0, http.StatusBadRequest},
		{"", 0, http.StatusOK}, // bare /fi and /fi/search mean page 0
		{"abc", 0, http.StatusBadRequest},
		{" 1", 0, http.StatusBadRequest},
	}
	for _, tc := range cases {
		page, status := parsePage(tc.input)
		if page != tc.wantPage || status != tc.wantStatus {
			t.Errorf("parsePage(%q) = (%d, %d), want (%d, %d)",
				tc.input, page, status, tc.wantPage, tc.wantStatus)
		}
	}
}

package render

import (
	"os"
	"testing"
)

// TestNewRenderParsesTemplates verifies that every template in public/html
// parses and that manifest.json can be loaded. The asset helper resolves
// paths relative to the working directory, so chdir to the module root.
func TestNewRenderParsesTemplates(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	// Mongo is only touched when a page is rendered, not at construction.
	r, err := NewRender(nil)
	if err != nil {
		t.Fatalf("NewRender failed: %v", err)
	}

	names := []string{"index_fi", "index_en", "search_fi", "search_en",
		"category_fi", "category_en", "source_fi", "source_en", "login"}
	for _, name := range names {
		if r.t.templates.Lookup(name) == nil {
			t.Errorf("template %q not found", name)
		}
	}
}

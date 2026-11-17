package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

// moduleRoot chdirs to the repository root so the real project assets can be
// read with their configured paths.
func moduleRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func readConcat(t *testing.T, paths []string) []byte {
	t.Helper()
	var out strings.Builder
	for _, p := range paths {
		content, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		out.Write(content)
		out.WriteByte('\n')
	}
	return []byte(out.String())
}

// TestProjectCssKeepsEveryLiveSelector is the safety property for the purge:
// any class or id that appears in the templates or scripts must still exist in
// the purged stylesheet. A false negative here means a broken page in
// production, so this test is the reason cmd/assets exists.
// hasSelector reports whether css contains a .name or #name selector that is
// not a prefix of a longer name (so ".menu" does not match ".menu-item").
func hasSelector(sheet []byte, word string) bool {
	for i := 0; i+1 < len(sheet); i++ {
		if sheet[i] != '.' && sheet[i] != '#' {
			continue
		}
		start := i + 1
		end := start + len(word)
		if end > len(sheet) || string(sheet[start:end]) != word {
			continue
		}
		if end < len(sheet) && isNameByte(sheet[end]) {
			continue
		}
		return true
	}
	return false
}

func TestProjectCssKeepsEveryLiveSelector(t *testing.T) {
	moduleRoot(t)

	cfg, err := loadConfig("assets.json")
	if err != nil {
		t.Fatal(err)
	}
	used, err := usedNames(cfg.UsedDirs)
	if err != nil {
		t.Fatal(err)
	}

	var cssSources []string
	for _, a := range cfg.Assets {
		if a.Type == "css" {
			cssSources = a.Sources
		}
	}
	if len(cssSources) == 0 {
		t.Fatal("no css asset configured")
	}
	input := readConcat(t, cssSources)
	purged := PurgeCSS(input, used)

	// Compare selector positions only: values such as url(x.css) or #000 would
	// otherwise look like the selectors ".css" and "#000".
	inSelectors := strings.Join(selectorTexts(input), "\n")
	outSelectors := strings.Join(selectorTexts(purged), "\n")

	var missing []string
	for word := range used {
		if hasSelector([]byte(inSelectors), word) && !hasSelector([]byte(outSelectors), word) {
			missing = append(missing, word)
		}
	}
	if len(missing) > 0 {
		t.Errorf("purge dropped %d selectors still referenced by templates/scripts: %v",
			len(missing), missing)
	}
	if len(purged) >= len(input) {
		t.Errorf("purge removed nothing: %d -> %d bytes", len(input), len(purged))
	}
	if opens, closes := strings.Count(string(purged), "{"), strings.Count(string(purged), "}"); opens != closes {
		t.Errorf("purged css has unbalanced braces: %d open, %d close", opens, closes)
	}
}

// TestProjectOutputIsNotEmptied guards the failure mode that bit the old shell
// pipeline: a minifier that silently produced 0 bytes of css and js.
func TestProjectOutputIsNotEmptied(t *testing.T) {
	moduleRoot(t)
	cfg, err := loadConfig("assets.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range cfg.Assets {
		content := readConcat(t, a.Sources)
		if len(content) < 100 {
			t.Errorf("%s: suspiciously small input (%d bytes)", a.Key, len(content))
		}
	}
}

// TestCommittedManifestIsConsistent runs the same verification as `-check`
// against the committed manifest, so stale or corrupted artifacts fail the test
// suite instead of only failing in CI. Nothing is written: the assets are built
// in memory and compared to what is on disk.
func TestCommittedManifestIsConsistent(t *testing.T) {
	moduleRoot(t)
	if _, err := os.Stat("manifest.json"); err != nil {
		t.Skip("manifest.json not built yet")
	}
	cfg, err := loadConfig("assets.json")
	if err != nil {
		t.Fatal(err)
	}
	used, err := usedNames(cfg.UsedDirs)
	if err != nil {
		t.Fatal(err)
	}

	minifier := minify.New()
	minifier.Add("text/css", &css.Minifier{})
	minifier.Add("text/javascript", &js.Minifier{})
	for _, a := range cfg.Assets {
		if err := a.build(minifier, used, true, true); err != nil {
			t.Fatalf("build %s: %v", a.Key, err)
		}
	}
	if err := checkManifest("manifest.json", buildManifest(cfg.Assets)); err != nil {
		t.Error(err)
	}
}

func TestCheckManifestRejectsCorruptedArtifacts(t *testing.T) {
	dir := t.TempDir()
	content := []byte("body{color:red}")
	good := filepath.Join(dir, "uutispuro-"+hashOf(content)+".min.css")
	want := []byte(`{"public/css/uutispuro.css":"` + filepath.ToSlash(good) + `"}` + "\n")

	t.Run("missing", func(t *testing.T) {
		manifest := filepath.Join(dir, "manifest.json")
		if err := os.WriteFile(manifest, want, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := checkManifest(manifest, want); err == nil {
			t.Error("missing artifact accepted")
		}
	})

	t.Run("empty", func(t *testing.T) {
		if err := os.WriteFile(good, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := checkManifest(filepath.Join(dir, "manifest.json"), want); err == nil {
			t.Error("empty artifact accepted")
		}
	})

	t.Run("tampered", func(t *testing.T) {
		if err := os.WriteFile(good, []byte("body{color:blue}"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := checkManifest(filepath.Join(dir, "manifest.json"), want); err == nil {
			t.Error("artifact that does not match its fingerprint accepted")
		}
	})

	t.Run("intact", func(t *testing.T) {
		if err := os.WriteFile(good, content, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := checkManifest(filepath.Join(dir, "manifest.json"), want); err != nil {
			t.Errorf("intact artifact rejected: %v", err)
		}
	})
}

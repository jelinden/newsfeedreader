// Command assets builds the site's minified css/js and writes manifest.json
// for go-asset-helper. It replaces the old purifycss + yuicompressor shell
// script, so the only toolchain needed is the Go compiler itself.
//
//	go run ./cmd/assets            # build assets and rewrite manifest.json
//	go run ./cmd/assets -check     # fail if manifest.json is out of date
//	go run ./cmd/assets -purge=false
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

const hashLength = 10

type asset struct {
	// Key is the path templates use, e.g. public/css/uutispuro.css.
	Key     string   `json:"key"`
	Type    string   `json:"type"` // "css" or "js"
	Sources []string `json:"sources"`

	input   int `json:"-"`
	purged  int `json:"-"`
	output  int `json:"-"`
	gzipped int `json:"-"`
	stale   int `json:"-"`
	path    string
}

type config struct {
	// UsedDirs are scanned for class/id names when purging css.
	UsedDirs []string `json:"usedDirs"`
	Assets   []*asset `json:"assets"`
}

var (
	configPath   = flag.String("config", "assets.json", "asset pipeline config")
	manifestPath = flag.String("manifest", "manifest.json", "manifest written for go-asset-helper")
	purgeFlag    = flag.Bool("purge", true, "drop css rules whose selectors are not used")
	check        = flag.Bool("check", false, "verify manifest.json without writing anything")
	quiet        = flag.Bool("quiet", false, "suppress the size report")
)

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "assets:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	used, err := usedNames(cfg.UsedDirs)
	if err != nil {
		return err
	}

	minifier := minify.New()
	minifier.Add("text/css", &css.Minifier{})
	// Locals are renamed; top level names stay, so inline handlers keep working.
	minifier.Add("text/javascript", &js.Minifier{KeepVarNames: false})

	for _, a := range cfg.Assets {
		if err := a.build(minifier, used, *purgeFlag, *check); err != nil {
			return fmt.Errorf("%s: %w", a.Key, err)
		}
	}

	manifest := buildManifest(cfg.Assets)
	if *check {
		if !*quiet {
			report(cfg.Assets)
		}
		if err := checkManifest(*manifestPath, manifest); err != nil {
			return err
		}
		if !*quiet {
			fmt.Printf("assets: %s is up to date\n", *manifestPath)
		}
		return nil
	}
	if err := writeFile(*manifestPath, manifest); err != nil {
		return err
	}
	if !*quiet {
		report(cfg.Assets)
	}
	return nil
}

func loadConfig(path string) (*config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &config{}
	if err := json.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(cfg.Assets) == 0 {
		return nil, fmt.Errorf("no assets configured in %s", path)
	}
	for _, a := range cfg.Assets {
		if a.Key == "" || len(a.Sources) == 0 {
			return nil, fmt.Errorf("invalid asset entry %+v", a)
		}
		if a.Type != "css" && a.Type != "js" {
			return nil, fmt.Errorf("asset %s: unsupported type %q", a.Key, a.Type)
		}
	}
	return cfg, nil
}

// usedNames scans templates and scripts for words that may be referenced by a
// css class or id selector.
func usedNames(dirs []string) (map[string]bool, error) {
	var files [][]byte
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".min.js") || strings.HasSuffix(name, ".min.css") {
				continue // generated output must not feed the purge
			}
			switch filepath.Ext(name) {
			case ".html", ".tmpl", ".js":
				content, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					return nil, err
				}
				files = append(files, content)
			}
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no templates or scripts found in %v", dirs)
	}
	return BuildUsedSet(files), nil
}

func (a *asset) build(minifier *minify.M, used map[string]bool, doPurge, dryRun bool) error {
	var combined bytes.Buffer
	for _, source := range a.Sources {
		content, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		combined.Write(content)
		combined.WriteByte('\n')
	}
	a.input = combined.Len()
	if a.input == 0 {
		return fmt.Errorf("input is empty")
	}

	source := combined.Bytes()
	if doPurge && a.Type == "css" {
		source = PurgeCSS(source, used)
		a.purged = len(source)
		if a.purged*10 < a.input { // almost everything was dropped: suspicious
			return fmt.Errorf("purge kept only %d of %d bytes, refusing to publish", a.purged, a.input)
		}
	}

	mediatype := "text/css"
	if a.Type == "js" {
		mediatype = "text/javascript"
	}
	var minified bytes.Buffer
	if err := minifier.Minify(mediatype, &minified, bytes.NewReader(source)); err != nil {
		return fmt.Errorf("minify: %w", err)
	}
	if minified.Len() == 0 {
		return fmt.Errorf("minifier produced empty output from %d bytes", len(source))
	}
	a.output = minified.Len()
	a.gzipped = gzipLen(minified.Bytes())

	extension := filepath.Ext(a.Key)
	dir, base := filepath.Split(a.Key)
	stem := strings.TrimSuffix(base, extension)
	fingerprint := hashOf(minified.Bytes())
	a.path = filepath.Join(dir, fmt.Sprintf("%s-%s.min%s", stem, fingerprint, extension))

	if err := a.removeStale(dir, stem, extension, dryRun); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	return writeFile(a.path, minified.Bytes())
}

// removeStale deletes previously generated fingerprints of this asset so the
// public directory does not accumulate dead files.
func (a *asset) removeStale(dir, stem, extension string, dryRun bool) error {
	matches, err := filepath.Glob(filepath.Join(dir, stem+"-*.min"+extension))
	if err != nil {
		return err
	}
	for _, match := range matches {
		if match == a.path {
			continue
		}
		if !dryRun {
			if err := os.Remove(match); err != nil {
				return err
			}
		}
		a.stale++
	}
	return nil
}

func buildManifest(assets []*asset) []byte {
	manifest := make(map[string]string, len(assets))
	for _, a := range assets {
		manifest[filepath.ToSlash(a.Key)] = filepath.ToSlash(a.path)
	}
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil
	}
	return append(out, '\n')
}

// checkManifest fails when the committed manifest no longer matches the
// current sources, which makes asset drift detectable in CI.
func checkManifest(path string, want []byte) error {
	current, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s is missing, run: go run ./cmd/assets", path)
		}
		return err
	}
	if !jsonEqual(current, want) {
		return fmt.Errorf("%s is out of date, run: go run ./cmd/assets", path)
	}
	var entries map[string]string
	if err := json.Unmarshal(want, &entries); err != nil {
		return err
	}
	for key, built := range entries {
		content, err := os.ReadFile(built)
		if err != nil {
			return fmt.Errorf("%s points at %s which does not exist", key, built)
		}
		if len(content) == 0 {
			return fmt.Errorf("%s is empty, run: go run ./cmd/assets", built)
		}
		// The fingerprint in the name is the promise that the file holds this
		// exact content; a hand edit or a truncated copy breaks that promise.
		if !strings.Contains(filepath.Base(built), hashOf(content)) {
			return fmt.Errorf("%s does not match its fingerprint, run: go run ./cmd/assets", built)
		}
	}
	return nil
}

func jsonEqual(a, b []byte) bool {
	var left, right map[string]string
	if err := json.Unmarshal(a, &left); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &right); err != nil {
		return false
	}
	if len(left) != len(right) {
		return false
	}
	for k, v := range right {
		if left[k] != v {
			return false
		}
	}
	return true
}

func writeFile(path string, content []byte) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, content, 0o644)
}

func report(assets []*asset) {
	rows := [][]string{{"ASSET", "INPUT", "AFTER PURGE", "MINIFIED", "GZIP", "STALE REMOVED"}}
	for _, a := range assets {
		purged := "-"
		if a.purged > 0 {
			purged = humanBytes(a.purged)
		}
		rows = append(rows, []string{
			a.path, humanBytes(a.input), purged, humanBytes(a.output), humanBytes(a.gzipped), fmt.Sprintf("%d", a.stale),
		})
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			fmt.Printf("%-*s  ", widths[i], cell)
		}
		fmt.Println()
	}
}

// hashOf is the fingerprint used in generated filenames.
func hashOf(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])[:hashLength]
}

func gzipLen(content []byte) int {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return 0
	}
	if _, err := writer.Write(content); err != nil {
		return 0
	}
	if err := writer.Close(); err != nil {
		return 0
	}
	return buf.Len()
}

func humanBytes(n int) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1fM", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1fk", float64(n)/1024)
	default:
		return fmt.Sprintf("%db", n)
	}
}

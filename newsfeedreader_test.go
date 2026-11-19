package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/jelinden/newsfeedreader/app/render"
	"github.com/jelinden/newsfeedreader/app/service"
	"github.com/jelinden/newsfeedreader/app/tick"
	"github.com/labstack/echo/v4"
)

// newTestApp wires up routes with zero-value dependencies. No request made in
// these tests reaches the database.
func newTestApp(t *testing.T) *echo.Echo {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(".."); err != nil { // serve from module root so public/ resolves
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })

	e := echo.New()
	setupRoutes(e, &Application{
		Mongo:  &service.Mongo{},
		Render: &render.Render{},
		Tick:   tick.NewTick(nil),
	})
	return e
}

func newStaticRequest(t *testing.T, target string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func TestStaticAllowed(t *testing.T) {
	c, rec := newStaticRequest(t, "/public/js/uutispuro.js")
	c.SetParamNames("filePath", "fileName")
	c.SetParamValues("js", "uutispuro.js")

	if err := static(c); err != nil {
		t.Fatalf("static: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if cc := rec.Header().Get("Cache-Control"); cc == "" {
		t.Error("expected Cache-Control header on static assets")
	}
}

func TestStaticRejectsDisallowedDir(t *testing.T) {
	c, rec := newStaticRequest(t, "/public/html/login.html")
	c.SetParamNames("filePath", "fileName")
	c.SetParamValues("html", "login.html")

	if err := static(c); err != nil {
		t.Fatalf("static: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestStaticRejectsTraversal(t *testing.T) {
	for _, fileName := range []string{"../../go.mod", "..", ".", ".hidden"} {
		c, rec := newStaticRequest(t, "/public/js/x")
		c.SetParamNames("filePath", "fileName")
		c.SetParamValues("js", fileName)

		if err := static(c); err != nil {
			t.Fatalf("static(%q): %v", fileName, err)
		}
		if rec.Code != http.StatusNotFound {
			t.Errorf("fileName %q: status = %d, want %d", fileName, rec.Code, http.StatusNotFound)
		}
	}
}

// TestRegisteredRouteShapes guards against the group-prefix bug that
// registered every language page as "//fi" and made the whole site 404.
func TestRegisteredRouteShapes(t *testing.T) {
	e := newTestApp(t)

	registered := map[string]bool{}
	for _, r := range e.Routes() {
		if strings.Contains(r.Path, "//") {
			t.Errorf("route %q contains an empty path segment", r.Path)
		}
		registered[r.Method+" "+r.Path] = true
	}

	want := []string{
		"GET /",
		"GET /fi",
		"GET /en",
		"GET /fi/:page",
		"GET /en/:page",
		"GET /fi/login",
		"GET /fi/search",
		"GET /en/search/:page",
		"GET /fi/category/:category",
		"GET /en/category/:category/:page",
		"GET /fi/source/:source/:page",
		"GET /public/:filePath/:fileName",
		"GET /serviceworker.js",
		"GET /api/click/:id",  // the Image() fallback
		"POST /api/click/:id", // navigator.sendBeacon
		"GET /api/news",
		"GET /ws/:channel",
	}
	for _, w := range want {
		if !registered[w] {
			t.Errorf("expected route %q to be registered", w)
		}
	}
}

func TestRoutingEndToEnd(t *testing.T) {
	e := newTestApp(t)

	cases := []struct {
		target string
		want   int
	}{
		{"/", http.StatusFound},                               // language redirect, no DB
		{"/uutiset/fi", http.StatusMovedPermanently},          // legacy redirect
		{"/uutiset/en", http.StatusMovedPermanently},          // legacy redirect
		{"/fi/category/kotimaa", http.StatusMovedPermanently}, // -> /fi/category/kotimaa/0
		{"/public/html/login.html", http.StatusNotFound},      // template dir not served
		{"/ws/nonsense", http.StatusNotFound},                 // unknown channel
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.target, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("GET %s = %d, want %d", tc.target, rec.Code, tc.want)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if got := rec.Header().Get("Location"); got != "/en" {
		t.Errorf("Accept-Language en redirected to %q, want /en", got)
	}
}

func TestStaticFilesReachableThroughRouter(t *testing.T) {
	e := newTestApp(t)
	if _, err := os.Stat("public/js/uutispuro.js"); err != nil {
		t.Skip("static asset not present")
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/public/js/uutispuro.js", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET /public/js/uutispuro.js = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "max-age=") {
		t.Error("expected Cache-Control on static asset")
	}
}

// clickScript is the file the browser runs. The route table has to agree with it,
// and the only way to know that is to read the method out of the script itself.
const clickScript = "public/js/uutispuro.js"

// TestClickBeaconIsAccepted is the regression the November frontend rewrite shipped:
// counting a click moved from an XHR GET to navigator.sendBeacon, which is a POST,
// while the route stayed GET-only.
//
// Two things hid it. A route with per-route middleware - the /api group has the rate
// limiter - is reported by Echo's router as 404 instead of 405, so the endpoint
// looked as if it had been removed rather than reached with the wrong verb. And no
// browser shows the response of a beacon, so there was nothing on screen either.
func TestClickBeaconIsAccepted(t *testing.T) {
	// Read the script before newTestApp: it changes the working directory.
	methods := beaconMethods(t)
	e := newTestApp(t)

	if len(methods) == 0 {
		t.Fatal("found no click beacon call in " + clickScript + "; the frontend changed shape, update this test")
	}
	for _, method := range methods {
		// The id is deliberately invalid: the handler answers 400 before it touches
		// the database, so 400 is what "the route matched" looks like here.
		req := httptest.NewRequest(method, "/api/click/not-an-objectid", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Errorf("%s /api/click/:id: 404, but %s is how %s reports a click; register that verb on the route",
				method, method, clickScript)
			continue
		}
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s /api/click/:id: status = %d, want 400 from the id validation", method, rec.Code)
		}
	}
}

// beaconMethods returns the HTTP methods the frontend uses against the click
// endpoint, read out of the script itself so the route table cannot drift from it
// unnoticed again.
func beaconMethods(t *testing.T) []string {
	t.Helper()
	text := clickScriptSource(t)
	if !strings.Contains(text, "/api/click/") {
		t.Fatalf("%s does not call /api/click/", clickScript)
	}
	var methods []string
	if strings.Contains(text, "navigator.sendBeacon") {
		// sendBeacon always issues a POST.
		methods = append(methods, http.MethodPost)
	}
	if regexp.MustCompile(`img\.src\s*=\s*url`).MatchString(text) {
		// The fallback for browsers without it is an image request, which is a GET.
		methods = append(methods, http.MethodGet)
	}
	return methods
}

// TestFrontendOnlyCallsRegisteredAPIPaths keeps the script from calling an endpoint
// that does not exist, which is the same failure one typo away.
func TestFrontendOnlyCallsRegisteredAPIPaths(t *testing.T) {
	text := clickScriptSource(t)
	e := newTestApp(t)

	calls := map[string]bool{}
	for _, match := range regexp.MustCompile(`"/api/([a-z]+)/`).FindAllStringSubmatch(text, -1) {
		calls["/api/"+match[1]+"/"] = true
	}
	if len(calls) == 0 {
		t.Fatal("no /api/ calls found in " + clickScript)
	}

	prefixes := map[string]bool{}
	for _, r := range e.Routes() {
		prefixes[r.Path] = true
		// "/api/click/:id" is reached by any path that starts with "/api/click/".
		if i := strings.Index(r.Path, ":"); i > 0 {
			prefixes[r.Path[:i]] = true
		}
	}
	for call := range calls {
		if !prefixes[call] {
			t.Errorf("%s calls %s but no route is registered under it", clickScript, call)
		}
	}
}

func clickScriptSource(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(clickScript)
	if err != nil {
		t.Fatalf("read the frontend: %v", err)
	}
	return string(data)
}

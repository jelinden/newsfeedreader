package main

import (
	"bytes"
	"strings"
	"testing"
)

func usedSet(words ...string) map[string]bool {
	set := map[string]bool{}
	for _, w := range words {
		set[w] = true
	}
	return set
}

func purge(t *testing.T, css string, used ...string) string {
	t.Helper()
	out := string(PurgeCSS([]byte(css), usedSet(used...)))
	// Normalise so tests can assert on readable expectations regardless of how
	// tightly the rewriter joins selector, brace and block.
	out = strings.ReplaceAll(out, "{", " { ")
	out = strings.ReplaceAll(out, "}", " } ")
	return strings.Join(strings.Fields(out), " ")
}

func TestPurgeKeepsUsedDropsUnused(t *testing.T) {
	in := ".item { color: red; } .orphan { color: blue; }"
	if got := purge(t, in, "item"); got != ".item { color: red; }" {
		t.Errorf("got %q", got)
	}
}

func TestElementSelectorsAreAlwaysKept(t *testing.T) {
	in := "body { margin: 0 } a:hover { color: red } h1, h2 { line-height: 1.2 }"
	got := purge(t, in, "nothing")
	for _, want := range []string{"body", "a:hover", "h1,h2"} {
		if !strings.Contains(got, want) {
			t.Errorf("element selector %q was dropped: %q", want, got)
		}
	}
}

func TestCommaSelectorsPrunedIndividually(t *testing.T) {
	in := ".used, .unused { color: red }"
	if got := purge(t, in, "used"); got != ".used { color: red }" {
		t.Errorf("got %q", got)
	}
}

// A selector survives when *any* of its classes/ids is seen in the sources.
// Classes can be added at runtime by scripts we do not fully control, so the
// cost of keeping dead css is a few bytes; the cost of dropping live css is a
// broken page.
func TestDescendantAndCompoundSelectorsUseSafeMatching(t *testing.T) {
	in := ".menu .link { color: red } .menu.hidden { color: blue }"
	got := purge(t, in, "menu")
	if !strings.Contains(got, ".menu .link") || !strings.Contains(got, ".menu.hidden") {
		t.Errorf("expected rules kept because .menu is used, got %q", got)
	}
	if got := purge(t, in, "unrelated"); got != "" {
		t.Errorf("expected both rules dropped when no class matches, got %q", got)
	}
}

func TestMediaQueriesPurgedRecursively(t *testing.T) {
	in := "@media screen and (min-width: 30em) { .used { color: red } .gone { color: blue } }"
	if got := purge(t, in, "used"); got != "@media screen and (min-width: 30em) { .used { color: red } }" {
		t.Errorf("got %q", got)
	}
	// The whole wrapper disappears when nothing inside survives.
	if got := purge(t, in, "unrelated"); got != "" {
		t.Errorf("expected empty media block, got %q", got)
	}
}

func TestNestedAtRules(t *testing.T) {
	in := "@supports (display: grid) { @media print { .gone { color: red } } .keep { color: blue } }"
	if got := purge(t, in, "keep"); got != "@supports (display: grid) { .keep { color: blue } }" {
		t.Errorf("got %q", got)
	}
}

func TestNonRuleAtRulesPreservedVerbatim(t *testing.T) {
	in := `@charset "utf-8";
@import url("x.css");
@font-face { font-family: "X"; src: url(x.woff2) }
@keyframes spin { from { transform: rotate(0) } to { transform: rotate(360deg) } }
.orphan { color: red }`
	got := purge(t, in, "nothing")
	for _, want := range []string{`@charset "utf-8";`, `@import url("x.css");`, `@font-face`, `@keyframes spin`} {
		if !strings.Contains(got, want) {
			t.Errorf("%q missing from output %q", want, got)
		}
	}
	if strings.Contains(got, ".orphan") {
		t.Error("unused rule after at-rules was kept")
	}
}

func TestBracesAndCommasInsideStringsAndParens(t *testing.T) {
	in := `.used::after { content: "a}b{c"; } .unused { background: url("a.png?q=1,2") }
	.a[title="x,y"] { color: red }`
	// Assert on the raw output: the whitespace normaliser in purge() would
	// itself rewrite the braces inside the string literal.
	got := string(PurgeCSS([]byte(in), usedSet("used", "a")))
	if !strings.Contains(got, `content: "a}b{c"`) {
		t.Errorf("string containing braces was mangled: %q", got)
	}
	if strings.Contains(got, ".unused") {
		t.Error("unused rule kept")
	}
	if !strings.Contains(got, `[title="x,y"]`) {
		t.Errorf("attribute selector with comma was mangled: %q", got)
	}
}

func TestFunctionalPseudoClassArgsNotSplit(t *testing.T) {
	// :is(.a, .b) is one selector; the comma inside must not split it.
	in := ":is(.a, .b) { color: red }"
	if got := purge(t, in, "a"); got != ":is(.a, .b) { color: red }" {
		t.Errorf("expected selector kept when any of its classes is used, got %q", got)
	}
	if got := purge(t, in, "zzz"); got != "" {
		t.Errorf("expected selector dropped, got %q", got)
	}
}

func TestMatchingIsCaseInsensitive(t *testing.T) {
	in := ".PureButton { color: red }"
	if got := purge(t, in, "purebutton"); got != ".PureButton { color: red }" {
		t.Errorf("case-insensitive match failed, got %q", got)
	}
}

func TestIdsAndNameCharacters(t *testing.T) {
	in := `#news-container { color: red } .pure-form-stacked { color: blue } .with_underscore { color: green }`
	got := purge(t, in, "news-container", "pure-form-stacked")
	if !strings.Contains(got, "#news-container") || !strings.Contains(got, ".pure-form-stacked") {
		t.Errorf("hyphenated names dropped: %q", got)
	}
	if strings.Contains(got, "with_underscore") {
		t.Error("unused underscore selector kept")
	}
}

func TestBuildUsedSetFromTemplates(t *testing.T) {
	files := [][]byte{
		[]byte(`<div class="item new"><a class="pure-button pure-button-primary">x</a></div>`),
		[]byte(`el.setAttribute("class", "dynamic-name");`),
	}
	used := BuildUsedSet(files)
	for _, want := range []string{"item", "new", "pure-button", "pure-button-primary", "dynamic-name"} {
		if !used[want] {
			t.Errorf("%q not found in used set", want)
		}
	}
	if used["mostread"] {
		t.Error("class not present in sources reported as used")
	}
}

func TestPurgeDoesNotMutateItsInput(t *testing.T) {
	// parse.NewInputBytes writes a NUL past the end of the buffer it is given.
	// Feeding it our own slice corrupted nested @media rules by overwriting the
	// byte that followed them, so PurgeCSS must never touch the caller's bytes.
	in := []byte("@media screen { .a { color: red } } .b { color: blue }")
	original := append([]byte(nil), in...)
	used := usedSet("a", "b")

	first := PurgeCSS(in, used)
	if !bytes.Equal(in, original) {
		t.Errorf("input was modified: %q -> %q", original, in)
	}
	second := PurgeCSS(in, used)
	if !bytes.Equal(first, second) {
		t.Errorf("repeat calls differ:\n first:  %q\n second: %q", first, second)
	}
	if strings.Count(string(first), "{") != strings.Count(string(first), "}") {
		t.Errorf("unbalanced output: %q", first)
	}
}

func TestPurgeNestedMediaKeepsFollowingRules(t *testing.T) {
	in := []byte(`
		@media (min-width: 30em) { .inside { color: red } }
		.after { color: blue }
		@media (min-width: 40em) { .second { color: lime } }
		.last { color: navy }`)
	got := string(PurgeCSS(in, usedSet("inside", "after", "second", "last")))
	for _, want := range []string{".inside", ".after", ".second", ".last"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q missing after nested at-rules: %q", want, got)
		}
	}
	if strings.Count(got, "{") != strings.Count(got, "}") {
		t.Errorf("unbalanced output: %q", got)
	}
}

func TestPurgeIsIdempotentAndBalanced(t *testing.T) {
	in := "@media (min-width: 10px) { .a { color: red } .b { color: blue } } .c { color: lime }"
	once := string(PurgeCSS([]byte(in), usedSet("a", "c")))
	twice := string(PurgeCSS([]byte(once), usedSet("a", "c")))
	if once != twice {
		t.Errorf("not idempotent:\n  once:  %s\n  twice: %s", once, twice)
	}
	if strings.Count(once, "{") != strings.Count(once, "}") {
		t.Errorf("unbalanced braces in %q", once)
	}
}

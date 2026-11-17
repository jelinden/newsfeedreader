package main

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// containerAtRules wrap style rules, so their bodies must be purged too.
var containerAtRules = map[string]bool{
	"media":     true,
	"supports":  true,
	"container": true,
	"layer":     true,
	"scope":     true,
}

// rule is one top level (per brace level) css construct.
type rule struct {
	text     []byte // complete source, including block and trailing semicolon
	prelude  []byte // selector or at-rule header, trimmed
	block    []byte // content between braces; nil for statements
	isAtRule bool
}

// scanRules splits one brace level of a stylesheet into rules. It relies on the
// CSS lexer rather than string matching so that braces and semicolons inside
// strings, url() values and comments are never mistaken for structure.
func scanRules(src []byte) []rule {
	// parse.NewInputBytes terminates its buffer by writing a NUL one byte past
	// the end. Rule text handed out by this function is a subslice of that
	// buffer, so recursing into an inner brace level would otherwise overwrite
	// the closing "}" of the enclosing rule. Lex an owned copy instead.
	buf := make([]byte, len(src), len(src)+1)
	copy(buf, src)

	input := parse.NewInputBytes(buf)
	lex := css.NewLexer(input)

	var (
		rules   []rule
		start   = -1 // first meaningful token of the rule being read
		braceAt = -1 // offset of the "{" that closes the prelude
		depth   int  // brace nesting inside the rule being read
	)

	push := func(end int) {
		r := rule{text: buf[start:end], isAtRule: buf[start] == '@'}
		r.prelude = bytes.TrimSpace(buf[start:max(0, end-1)])
		if braceAt >= 0 {
			r.prelude = bytes.TrimSpace(buf[start:braceAt])
			r.block = buf[braceAt+1 : max(braceAt+1, end-1)]
		}
		rules = append(rules, r)
		start, braceAt = -1, -1
	}

	for {
		token, value := lex.Next()
		if token == css.ErrorToken {
			break
		}
		// The lexer collapses its cursor after every token, so Offset() is the
		// end of that token and the lexeme length gives its start.
		to := input.Offset()
		from := to - len(value)

		if token == css.WhitespaceToken || token == css.CommentToken {
			continue
		}
		if depth == 0 && start < 0 {
			start = from
		}

		switch token {
		case css.LeftBraceToken:
			if depth == 0 {
				braceAt = from
			}
			depth++
		case css.RightBraceToken:
			depth--
			if depth == 0 && start >= 0 {
				push(to)
			}
		case css.SemicolonToken:
			if depth == 0 && start >= 0 {
				push(to)
			}
		}
	}
	return rules
}

// PurgeCSS drops style rules whose class and id selectors never appear in
// used. Selectors made only of element names (body, a, h1) and at-rules whose
// body is not a rule list (@keyframes, @font-face, @import, ...) are always
// kept, because there is nothing safe to match against.
//
// Matching is deliberately conservative: keeping dead css costs a few bytes,
// removing live css breaks the page.
func PurgeCSS(src []byte, used map[string]bool) []byte {
	var out bytes.Buffer
	out.Grow(len(src))
	purgeInto(&out, src, used)
	return out.Bytes()
}

func purgeInto(out *bytes.Buffer, src []byte, used map[string]bool) {
	for _, r := range scanRules(src) {
		switch {
		case !r.isAtRule:
			if kept := keepSelectors(r.prelude, used); kept != nil {
				out.Write(kept)
				out.WriteByte('{')
				out.Write(r.block)
				out.WriteByte('}')
			}
		case containerAtRules[atRuleName(r.prelude)]:
			var inner bytes.Buffer
			purgeInto(&inner, r.block, used)
			if inner.Len() > 0 {
				out.Write(r.prelude)
				out.WriteByte('{')
				out.Write(inner.Bytes())
				out.WriteByte('}')
			}
		default:
			out.Write(r.text)
		}
	}
}

// selectorTexts returns every style rule selector in the stylesheet, including
// the ones nested inside @media and friends. Declaration values are excluded,
// which matters when checking class names against real css text.
func selectorTexts(src []byte) []string {
	var out []string
	for _, r := range scanRules(src) {
		switch {
		case !r.isAtRule:
			out = append(out, string(r.prelude))
		case containerAtRules[atRuleName(r.prelude)]:
			out = append(out, selectorTexts(r.block)...)
		}
	}
	return out
}

// keepSelectors returns the prelude with unused selector parts removed, or nil
// when the whole rule can be dropped.
func keepSelectors(prelude []byte, used map[string]bool) []byte {
	parts := splitSelectors(prelude)
	kept := make([][]byte, 0, len(parts))
	for _, part := range parts {
		trimmed := bytes.TrimSpace(part)
		if len(trimmed) == 0 {
			continue
		}
		if selectorIsUsed(trimmed, used) {
			kept = append(kept, trimmed)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return bytes.Join(kept, []byte(","))
}

// selectorIsUsed reports whether a single selector may still be reachable. It
// is kept when it carries no class or id at all, or when *any* of its
// classes/ids appears in the used set. A selector survives on the assumption
// that classes may be applied at runtime by code we do not scan.
func selectorIsUsed(selector []byte, used map[string]bool) bool {
	names := selectorNames(selector)
	if len(names) == 0 {
		return true
	}
	for _, name := range names {
		if used[strings.ToLower(name)] {
			return true
		}
	}
	return false
}

// selectorNames extracts class and id names from a selector.
func selectorNames(selector []byte) []string {
	var names []string
	for i := 0; i < len(selector); i++ {
		if selector[i] != '.' && selector[i] != '#' {
			continue
		}
		j := i + 1
		for j < len(selector) && isNameByte(selector[j]) {
			j++
		}
		if j > i+1 {
			names = append(names, string(selector[i+1:j]))
		}
		i = j - 1
	}
	return names
}

func isNameByte(c byte) bool {
	return c == '-' || c == '_' ||
		('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') ||
		('0' <= c && c <= '9') || c >= 0x80
}

func splitSelectors(prelude []byte) [][]byte {
	var (
		parts   [][]byte
		depth   int
		start   int
		inQuote byte
	)
	for i := 0; i < len(prelude); i++ {
		c := prelude[i]
		switch {
		case inQuote != 0:
			if c == inQuote && prelude[i-1] != '\\' {
				inQuote = 0
			}
		case c == '"' || c == '\'':
			inQuote = c
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
		case c == ',' && depth == 0:
			parts = append(parts, prelude[start:i])
			start = i + 1
		}
	}
	return append(parts, prelude[start:])
}

func atRuleName(prelude []byte) string {
	rest := strings.TrimPrefix(string(prelude), "@")
	end := strings.IndexFunc(rest, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '('
	})
	if end < 0 {
		return strings.ToLower(rest)
	}
	return strings.ToLower(rest[:end])
}

var wordPattern = regexp.MustCompile(`[\p{L}0-9_-]+`)

// BuildUsedSet collects every word occurring in the given source files
// (templates and scripts). Class and id selectors are matched against it;
// everything that appears anywhere wins, which is imprecise in the safe
// direction.
func BuildUsedSet(files [][]byte) map[string]bool {
	used := map[string]bool{}
	for _, content := range files {
		for _, word := range wordPattern.FindAll(content, -1) {
			used[strings.ToLower(string(word))] = true
		}
	}
	return used
}

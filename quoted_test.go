// Copyright (c) 2014 Alex Kalyvitis

package mustache

import (
	"reflect"
	"strings"
	"testing"
)

// TestParsePath exercises the quote-aware path splitter directly: barewords, both
// quote styles, escapes, empty keys, unicode, and every malformed-quote error.
func TestParsePath(t *testing.T) {
	for _, test := range []struct {
		name    string
		raw     string
		want    []pathSegment
		wantErr string // non-empty => parsePath must return an error containing this
	}{
		// Barewords (unchanged behavior).
		{"single", "x", []pathSegment{{"x", false}}, ""},
		{"dotted", "a.b", []pathSegment{{"a", false}, {"b", false}}, ""},
		{"dotted3", "a.b.c", []pathSegment{{"a", false}, {"b", false}, {"c", false}}, ""},
		{"empty middle", "a..b", []pathSegment{{"a", false}, {"", false}, {"b", false}}, ""},
		{"leading dot", ".x", []pathSegment{{"", false}, {"x", false}}, ""},
		{"trailing dot", "a.", []pathSegment{{"a", false}, {"", false}}, ""},
		{"lone dot", ".", []pathSegment{{".", false}}, ""},

		// Double-quoted.
		{"dq single", `"a"`, []pathSegment{{"a", true}}, ""},
		{"dq dotted key", `x."a.b"`, []pathSegment{{"x", false}, {"a.b", true}}, ""},
		{"dq all dots", `"a.b.c.d"`, []pathSegment{{"a.b.c.d", true}}, ""},
		{"dq then bareword", `"a.b".c`, []pathSegment{{"a.b", true}, {"c", false}}, ""},
		{"dq non-terminal", `a."b.c".d`, []pathSegment{{"a", false}, {"b.c", true}, {"d", false}}, ""},

		// Single-quoted (interchangeable with double quotes).
		{"sq single", `'a'`, []pathSegment{{"a", true}}, ""},
		{"sq dotted key", `x.'a.b'`, []pathSegment{{"x", false}, {"a.b", true}}, ""},
		{"sq then bareword", `'a.b'.c`, []pathSegment{{"a.b", true}, {"c", false}}, ""},
		{"sq non-terminal", `a.'b.c'.d`, []pathSegment{{"a", false}, {"b.c", true}, {"d", false}}, ""},

		// Mixed styles in one path.
		{"mixed", `'a'."b".'c'`, []pathSegment{{"a", true}, {"b", true}, {"c", true}}, ""},

		// Empty quoted keys are legal (empty-string key).
		{"dq empty", `""`, []pathSegment{{"", true}}, ""},
		{"sq empty", `''`, []pathSegment{{"", true}}, ""},
		{"dq empty nested", `x.""`, []pathSegment{{"x", false}, {"", true}}, ""},
		{"sq empty middle", `a.''.b`, []pathSegment{{"a", false}, {"", true}, {"b", false}}, ""},

		// Escapes inside double quotes: \\ and \".
		{"dq esc quote", `"a\"b"`, []pathSegment{{`a"b`, true}}, ""},
		{"dq esc quote dot", `"a\".b"`, []pathSegment{{`a".b`, true}}, ""},
		{"dq esc backslash", `"a\\b"`, []pathSegment{{`a\b`, true}}, ""},
		{"dq lone backslash", `"\\"`, []pathSegment{{`\`, true}}, ""},

		// Escapes inside single quotes: \\ and \'.
		{"sq esc quote", `'a\'b'`, []pathSegment{{`a'b`, true}}, ""},
		{"sq esc backslash", `'a\\b'`, []pathSegment{{`a\b`, true}}, ""},

		// The other quote char is an ordinary literal inside a quoted key.
		{"dq contains sq", `"a'b"`, []pathSegment{{`a'b`, true}}, ""},
		{"sq contains dq", `'a"b'`, []pathSegment{{`a"b`, true}}, ""},

		// Whitespace inside quotes is preserved.
		{"interior space", `"a b"`, []pathSegment{{"a b", true}}, ""},
		{"trailing space", `"a "`, []pathSegment{{"a ", true}}, ""},
		{"leading space", `"  a"`, []pathSegment{{"  a", true}}, ""},
		{"lone space", `" "`, []pathSegment{{" ", true}}, ""},

		// Unicode / multibyte keys (byte-scan safety for '.' and quotes).
		{"unicode", `"café.au"`, []pathSegment{{"café.au", true}}, ""},
		{"cjk", `"日本.語"`, []pathSegment{{"日本.語", true}}, ""},

		// A quoted "." is a literal key, not the whole-context shortcut.
		{"dq dot key", `"."`, []pathSegment{{".", true}}, ""},
		{"sq dot key", `'.'`, []pathSegment{{".", true}}, ""},

		// Quoted segment followed by a trailing dot (empty final segment).
		{"dq trailing dot", `"a".`, []pathSegment{{"a", true}, {"", false}}, ""},
		{"sq trailing dot", `'a'.`, []pathSegment{{"a", true}, {"", false}}, ""},
		// A quote that is not at a segment start is an ordinary character.
		{"quote mid bareword", `a"b`, []pathSegment{{`a"b`, false}}, ""},

		// Errors: unterminated quotes.
		{"unterminated dq", `"foo`, nil, "unterminated quote"},
		{"unterminated sq", `'foo`, nil, "unterminated quote"},
		{"unterminated nested", `x."foo`, nil, "unterminated quote"},
		{"unterminated with dot", `"foo.bar`, nil, "unterminated quote"},
		// Errors: dangling / unknown escapes.
		{"dangling escape", `"a\`, nil, "dangling escape"},
		{"unknown escape dq", `"a\q"`, nil, "invalid escape"},
		{"unknown escape sq", `'a\z'`, nil, "invalid escape"},
		{"dq escape sq invalid", `"a\'"`, nil, "invalid escape"},
		{"sq escape dq invalid", `'a\"'`, nil, "invalid escape"},
		// Errors: junk after a closing quote.
		{"junk after dq", `"a"b`, nil, "unexpected"},
		{"junk after sq", `'a'b`, nil, "unexpected"},
		{"adjacent quotes", `"a""b"`, nil, "unexpected"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parsePath(test.raw)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("parsePath(%q): want error containing %q, got %v", test.raw, test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePath(%q): unexpected error %v", test.raw, err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("parsePath(%q) = %#v, want %#v", test.raw, got, test.want)
			}
		})
	}
}

// TestLookupQuoted exercises lookupPath over parsed quoted paths, including the
// bug-prone cases: non-terminal quoted segments, quoted "." vs whole-context,
// dotted struct tags, empty keys, and found-but-falsy values.
func TestLookupQuoted(t *testing.T) {
	type tagged struct {
		Field string `mustache:"a.b"`
	}
	for _, test := range []struct {
		name      string
		path      string
		context   interface{}
		wantValue interface{}
		wantTruth bool
	}{
		{
			"double-quoted dotted key",
			`"a.b"`,
			map[string]interface{}{"a.b": "v"},
			"v", true,
		},
		{
			"single-quoted dotted key",
			`'a.b'`,
			map[string]interface{}{"a.b": "v"},
			"v", true,
		},
		{
			"nested quoted terminal",
			`attributes."deployment.environment.name"`,
			map[string]interface{}{"attributes": map[string]interface{}{
				"deployment.environment.name": "prod"}},
			"prod", true,
		},
		{
			"non-terminal quoted (double)",
			`a."b.c".d`,
			map[string]interface{}{"a": map[string]interface{}{
				"b.c": map[string]interface{}{"d": "X"}}},
			"X", true,
		},
		{
			"non-terminal quoted (single)",
			`a.'b.c'.d`,
			map[string]interface{}{"a": map[string]interface{}{
				"b.c": map[string]interface{}{"d": "X"}}},
			"X", true,
		},
		{
			"quoted dot is a literal key",
			`"."`,
			map[string]interface{}{".": "dot-key"},
			"dot-key", true,
		},
		{
			"bare dot is whole context",
			`.`,
			"whole",
			"whole", true,
		},
		{
			"dotted struct tag via quoted key",
			`"a.b"`,
			tagged{Field: "tagged"},
			"tagged", true,
		},
		{
			"empty quoted key",
			`""`,
			map[string]interface{}{"": "empty-key"},
			"empty-key", true,
		},
		{
			"found but falsy int",
			`n`,
			map[string]interface{}{"n": 0},
			0, false,
		},
		{
			"found but falsy bool",
			`b`,
			map[string]interface{}{"b": false},
			false, false,
		},
		{
			"missing quoted key",
			`x."a.b"`,
			map[string]interface{}{"x": map[string]interface{}{"other": 1}},
			nil, false,
		},
		{
			"found but falsy empty string",
			`s`,
			map[string]interface{}{"s": ""},
			"", false,
		},
		{
			"falsy intermediate stops walk",
			`a.b.c`,
			map[string]interface{}{"a": map[string]interface{}{"b": 0}},
			nil, false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, truth := lookupPath(mustPath(test.path), test.context)
			if !reflect.DeepEqual(value, test.wantValue) {
				t.Errorf("lookupPath(%q) value = %#v, want %#v", test.path, value, test.wantValue)
			}
			if truth != test.wantTruth {
				t.Errorf("lookupPath(%q) truth = %t, want %t", test.path, truth, test.wantTruth)
			}
		})
	}
}

// TestQuotedRender is the end-to-end coverage: templates with quoted keys rendered
// against real contexts.
func TestQuotedRender(t *testing.T) {
	for _, test := range []struct {
		name     string
		template string
		context  interface{}
		want     string
		options  []Option
	}{
		{
			"double quotes",
			`{{ x."a.b" }}`,
			map[string]interface{}{"x": map[string]interface{}{"a.b": "v"}},
			"v", nil,
		},
		{
			"single quotes",
			`{{ x.'a.b' }}`,
			map[string]interface{}{"x": map[string]interface{}{"a.b": "v"}},
			"v", nil,
		},
		{
			"nested dotted key",
			`{{ attributes."deployment.environment.name" }}`,
			map[string]interface{}{"attributes": map[string]interface{}{
				"deployment.environment.name": "prod"}},
			"prod", nil,
		},
		{
			"interior space",
			`{{ x."a b" }}`,
			map[string]interface{}{"x": map[string]interface{}{"a b": "sp"}},
			"sp", nil,
		},
		{
			"escaped quote in key",
			`{{ x."a\".b" }}`,
			map[string]interface{}{"x": map[string]interface{}{`a".b`: "esc"}},
			"esc", nil,
		},
		{
			"non-terminal quoted segment",
			`{{ a."b.c".d }}`,
			map[string]interface{}{"a": map[string]interface{}{
				"b.c": map[string]interface{}{"d": "X"}}},
			"X", nil,
		},
		{
			"html escaped by default",
			`{{ x."a.b" }}`,
			map[string]interface{}{"x": map[string]interface{}{"a.b": "<b>"}},
			"&lt;b&gt;", nil,
		},
		{
			"triple brace is raw",
			`{{{ x."a.b" }}}`,
			map[string]interface{}{"x": map[string]interface{}{"a.b": "<b>"}},
			"<b>", nil,
		},
		{
			"ampersand raw alt",
			`{{& x."a.b" }}`,
			map[string]interface{}{"x": map[string]interface{}{"a.b": "<b>"}},
			"<b>", nil,
		},
		{
			"empty quoted key",
			`{{ x."" }}`,
			map[string]interface{}{"x": map[string]interface{}{"": "E"}},
			"E", nil,
		},
		{
			"back-compat: unquoted-inside resolves stripped key",
			`{{ x."a" }}`,
			map[string]interface{}{"x": map[string]interface{}{"a": "plain", `"a"`: "old"}},
			"plain", nil,
		},
		{
			"section with quoted name",
			`{{#a."b.c"}}{{d}}{{/a."b.c"}}`,
			map[string]interface{}{"a": map[string]interface{}{
				"b.c": map[string]interface{}{"d": "in"}}},
			"in", nil,
		},
		{
			"inverse section with quoted name",
			`{{^a."b.c"}}no{{/a."b.c"}}`,
			map[string]interface{}{"a": map[string]interface{}{"b.c": false}},
			"no", nil,
		},
		{
			"custom delimiters: key with }} is addressable",
			`<% x."a}}b" %>`,
			map[string]interface{}{"x": map[string]interface{}{"a}}b": "delim"}},
			"delim", []Option{Delimiters("<%", "%>")},
		},
		{
			"unicode key rendered",
			`{{ x."café.au" }}`,
			map[string]interface{}{"x": map[string]interface{}{"café.au": "u"}},
			"u", nil,
		},
		{
			"escaped backslash key rendered",
			`{{ x."a\\b" }}`,
			map[string]interface{}{"x": map[string]interface{}{`a\b`: "bs"}},
			"bs", nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpl := New(test.options...)
			if err := tmpl.ParseString(test.template); err != nil {
				t.Fatalf("ParseString(%q): %v", test.template, err)
			}
			got, err := tmpl.RenderString(test.context)
			if err != nil {
				t.Fatalf("RenderString: %v", err)
			}
			if got != test.want {
				t.Errorf("template %q => %q, want %q", test.template, got, test.want)
			}
		})
	}
}

// TestQuotedParseErrors asserts that malformed quotes are parse errors surfaced
// unconditionally — regardless of the SilentMiss setting (which only governs missing
// values at render time).
func TestQuotedParseErrors(t *testing.T) {
	for _, test := range []struct {
		name     string
		template string
		wantErr  string
	}{
		{"unterminated", `{{ "foo }}`, "unterminated quote"},
		{"unterminated nested", `{{ x."foo.bar }}`, "unterminated quote"},
		{"junk after close", `{{ x."a"b }}`, "unexpected"},
		{"unknown escape", `{{ x."a\q" }}`, "invalid escape"},
		{"delimiter inside key truncates", `{{ x."a}}b" }}`, "unterminated quote"},
		{"section quote-style mismatch", `{{#a."b"}}x{{/a.'b'}}`, "closing tag"},
		{"raw tag malformed", `{{{ x."a }}}`, "unterminated quote"},
		{"section name malformed", `{{#x."a }}y{{/x}}`, "unterminated quote"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, silent := range []bool{true, false} {
				tmpl := New(SilentMiss(silent))
				err := tmpl.ParseString(test.template)
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Errorf("SilentMiss(%t) ParseString(%q): want error containing %q, got %v",
						silent, test.template, test.wantErr, err)
				}
			}
		})
	}
}

// TestQuotedMissVsParseError shows the distinction: a well-formed quoted path that
// simply misses is governed by SilentMiss (parse succeeds either way), unlike a
// malformed quote which fails at parse time.
func TestQuotedMissVsParseError(t *testing.T) {
	const tmplStr = `{{ x."a.b" }}`

	// SilentMiss(true): parses, renders empty, no error.
	silent := New(SilentMiss(true))
	if err := silent.ParseString(tmplStr); err != nil {
		t.Fatalf("SilentMiss(true) ParseString: %v", err)
	}
	got, err := silent.RenderString(map[string]interface{}{})
	if err != nil {
		t.Fatalf("SilentMiss(true) RenderString: unexpected error %v", err)
	}
	if got != "" {
		t.Errorf("SilentMiss(true) render = %q, want empty", got)
	}

	// SilentMiss(false): parses fine, but the miss surfaces as a render error.
	loud := New(SilentMiss(false))
	if err := loud.ParseString(tmplStr); err != nil {
		t.Fatalf("SilentMiss(false) ParseString: %v", err)
	}
	if _, err := loud.RenderString(map[string]interface{}{}); err == nil {
		t.Errorf("SilentMiss(false) render: expected a lookup error, got nil")
	}
}

// TestQuotedTestValue covers quoted keys inside a {{#test_value}} identifier — both
// a successful resolution and the malformed-quote parse error on that path.
func TestQuotedTestValue(t *testing.T) {
	tmpl := New(TestValueSection())
	if err := tmpl.ParseString(`{{#test_value {{x."a.b"}} "v"}}hit{{/test_value}}`); err != nil {
		t.Fatalf("ParseString: %v", err)
	}
	got, err := tmpl.RenderString(map[string]interface{}{"x": map[string]interface{}{"a.b": "v"}})
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if got != "hit" {
		t.Errorf("test_value render = %q, want %q", got, "hit")
	}

	bad := New(TestValueSection())
	if err := bad.ParseString(`{{#test_value {{x."a}} "v"}}hit{{/test_value}}`); err == nil ||
		!strings.Contains(err.Error(), "unterminated quote") {
		t.Errorf("malformed test_value ident: want unterminated quote error, got %v", err)
	}
}

// TestLookupPathEmpty covers lookupPath's defensive guard for an empty path.
func TestLookupPathEmpty(t *testing.T) {
	ctx := map[string]interface{}{"a": 1}
	if v, ok := lookupPath(nil, ctx); v != nil || ok {
		t.Errorf("lookupPath(nil) = (%v, %t), want (nil, false)", v, ok)
	}
	if v, ok := lookupPath([]pathSegment{}, ctx); v != nil || ok {
		t.Errorf("lookupPath([]) = (%v, %t), want (nil, false)", v, ok)
	}
}

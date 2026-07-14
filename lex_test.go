// Copyright (c) 2014 Alex Kalyvitis

package mustache

import "testing"

func TestLexer(t *testing.T) {
	for _, test := range []struct {
		template string
		expected []token
	}{
		{
			"foo {{{bar}}}\nbaz {{! this is ignored }}",
			[]token{
				{typ: tokenText, val: "foo "},
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenRawStart, val: "{"},
				{typ: tokenIdentifier, val: "bar"},
				{typ: tokenRawEnd, val: "}"},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenText, val: "\nbaz "},
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenComment, val: "!"},
				{typ: tokenText, val: " this is ignored "},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenEOF},
			},
		},
		{
			"\nfoo {{bar}} baz {{=| |=}}\r\n |foo| |={{! !}}=| {{!bar!}}",
			[]token{
				{typ: tokenText, val: "\nfoo "},
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenIdentifier, val: "bar"},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenText, val: " baz "},
				{typ: tokenSetDelim},
				{typ: tokenText, val: "\r\n "},
				{typ: tokenLeftDelim, val: "|"},
				{typ: tokenIdentifier, val: "foo"},
				{typ: tokenRightDelim, val: "|"},
				{typ: tokenText, val: " "},
				{typ: tokenSetDelim},
				{typ: tokenText, val: " "},
				{typ: tokenLeftDelim, val: "{{!"},
				{typ: tokenIdentifier, val: "bar"},
				{typ: tokenRightDelim, val: "!}}"},
				{typ: tokenEOF},
			},
		},
		{
			"foo {{bar.with space}}\n{{ trim }}",
			[]token{
				{typ: tokenText, val: "foo "},
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenIdentifier, val: "bar.with space"},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenText, val: "\n"},
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenIdentifier, val: "trim"},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenEOF},
			},
		},
		{
			"foo {{#test_value {{foo}} \"bar\"}}{{/test_value}}",
			[]token{
				{typ: tokenText, val: "foo "},
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenTestValue, val: "#"},
				{typ: tokenIdentifier, val: "test_value"},
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenIdentifier, val: "foo"},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenText, val: "bar"},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenSectionEnd, val: "/"},
				{typ: tokenIdentifier, val: "test_value"},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenEOF},
			},
		},
		{
			// A quoted key is emitted as a single identifier token, quotes and
			// interior dots included; surrounding whitespace is trimmed.
			`foo {{ metrics."http.request.count" }}`,
			[]token{
				{typ: tokenText, val: "foo "},
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenIdentifier, val: `metrics."http.request.count"`},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenEOF},
			},
		},
		{
			// Single quotes and a quoted segment in the middle of a path.
			`{{ fields.'service.name'.value }}`,
			[]token{
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenIdentifier, val: `fields.'service.name'.value`},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenEOF},
			},
		},
		{
			// A single '}' inside a quoted key is not the closing delimiter.
			`{{ x."a}b" }}`,
			[]token{
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenIdentifier, val: `x."a}b"`},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenEOF},
			},
		},
		{
			// A backslash-escaped quote is carried through verbatim to the parser.
			`{{ x."a\".b" }}`,
			[]token{
				{typ: tokenLeftDelim, val: "{{"},
				{typ: tokenIdentifier, val: `x."a\".b"`},
				{typ: tokenRightDelim, val: "}}"},
				{typ: tokenEOF},
			},
		},
	} {
		var (
			lexer = newLexer(test.template, "{{", "}}", true)
			token = lexer.token()
			i     = 0
		)
		for token.typ > tokenEOF {
			t.Logf("%s\n", token)
			if i >= len(test.expected) {
				t.Fatalf("token stream exceeded the length of expected tokens.")
			}
			if token.typ != test.expected[i].typ {
				t.Errorf("unexpected token %q, expected %q", token.typ, test.expected[i].typ)
			}
			if token.val != test.expected[i].val {
				t.Errorf("unexpected value %q, expected %q", token.val, test.expected[i].val)
			}
			token = lexer.token()
			i++
		}
	}
}

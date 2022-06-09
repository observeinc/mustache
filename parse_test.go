// Copyright (c) 2014 Alex Kalyvitis

package mustache

import (
	"reflect"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	for _, test := range []struct {
		template string
		expected []node
	}{
		{
			"{{#foo}}\n\t{{#foo}}hello nested{{/foo}}{{/foo}}",
			[]node{
				&sectionNode{"foo", false, []node{
					textNode("\n\t"),
					&sectionNode{"foo", false, []node{
						textNode("hello nested"),
					}},
				}},
			},
		},
		{
			"\nfoo {{bar}} {{#alex}}\r\n\tbaz\n{{/alex}} {{!foo}}",
			[]node{
				textNode("\nfoo "),
				&varNode{"bar", htmlEscape},
				textNode(" "),
				&sectionNode{"alex", false, []node{
					textNode("\r\n\tbaz\n"),
				}},
				textNode(" "),
				commentNode("foo"),
			},
		},
		{
			"this will{{^foo}}not{{/foo}} be rendered",
			[]node{
				textNode("this will"),
				&sectionNode{"foo", true, []node{
					textNode("not"),
				}},
				textNode(" be rendered"),
			},
		},
		{
			"{{#list}}({{.}}){{/list}}",
			[]node{
				&sectionNode{"list", false, []node{
					textNode("("),
					&varNode{".", htmlEscape},
					textNode(")"),
				}},
			},
		},
		{
			"{{#*}}({{.}}){{/*}}",
			[]node{
				&sectionNode{"*", false, []node{
					textNode("("),
					&varNode{".", htmlEscape},
					textNode(")"),
				}},
			},
		},
		{
			"{{#list}}({{*}}){{/list}}",
			[]node{
				&sectionNode{"list", false, []node{
					textNode("("),
					&varNode{"*", htmlEscape},
					textNode(")"),
				}},
			},
		},
		{
			"{{#test_value {{foo}} \"bar\"}}({{a}a}}){{/test_value}}",
			[]node{
				&testNode{"foo", "bar", []node{
					textNode("("),
					&varNode{"a}a", htmlEscape},
					textNode(")"),
				}},
			},
		},
		{
			"{{#test_value {{foo}} \"bar\"}}{{#a}}{{b}}{{/a}}{{/test_value}}",
			[]node{
				&testNode{"foo", "bar", []node{
					&sectionNode{"a", false, []node{
						&varNode{"b", htmlEscape},
					}},
				}},
			},
		},
		{
			"{{#list}}({{a}a}}){{/list}}",
			[]node{
				&sectionNode{"list", false, []node{
					textNode("("),
					&varNode{"a}a", htmlEscape},
					textNode(")"),
				}},
			},
		},
	} {
		parser := newParser(newLexer(test.template, "{{", "}}", true), htmlEscape)
		elems, err := parser.parse()
		if err != nil {
			t.Fatal(err)
		}
		for i, elem := range elems {
			if !reflect.DeepEqual(elem, test.expected[i]) {
				t.Errorf("elements are not equal %v != %v", elem, test.expected[i])
			}
		}
	}
}

func TestParserNegative(t *testing.T) {
	for _, test := range []struct {
		template string
		expErr   string
	}{
		{
			"{{foo}",
			`1:6 syntax error: unreachable code t_error:"unclosed tag"`,
		},
		{
			"{{#test_value {{a}} b}}",
			`1:21 syntax error: unexpected token t_error:"invalid test_value value token"`,
		},
		{
			"{{#test_value {{a}} \"b}}",
			`1:24 syntax error: unexpected token t_error:"failed to find close \" for test_value value token"`,
		},
		{
			"{{#test_value {{a}} \"b\"}}",
			`token "t_ident" not found`,
		},
		{
			"{{#test_value a b}}",
			`1:14 syntax error: unexpected token t_error:"Missing test_value identifier"`,
		},
		{
			"{{#test_value {{a b\"}}",
			`1:22 syntax error: unexpected token t_error:"invalid test_value value token"`,
		},
	} {
		parser := newParser(newLexer(test.template, "{{", "}}", true), htmlEscape)
		_, err := parser.parse()
		if err == nil || !strings.Contains(err.Error(), test.expErr) {
			t.Errorf("expect error: %q, got %q", test.expErr, err)
		}
	}
}

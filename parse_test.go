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
				&sectionNode{"foo", mustPath("foo"), false, []node{
					textNode("\n\t"),
					&sectionNode{"foo", mustPath("foo"), false, []node{
						textNode("hello nested"),
					}},
				}},
			},
		},
		{
			"\nfoo {{bar}} {{#alex}}\r\n\tbaz\n{{/alex}} {{!foo}}",
			[]node{
				textNode("\nfoo "),
				&varNode{"bar", mustPath("bar"), htmlEscape},
				textNode(" "),
				&sectionNode{"alex", mustPath("alex"), false, []node{
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
				&sectionNode{"foo", mustPath("foo"), true, []node{
					textNode("not"),
				}},
				textNode(" be rendered"),
			},
		},
		{
			"{{#list}}({{.}}){{/list}}",
			[]node{
				&sectionNode{"list", mustPath("list"), false, []node{
					textNode("("),
					&varNode{".", mustPath("."), htmlEscape},
					textNode(")"),
				}},
			},
		},
		{
			"{{#*}}({{.}}){{/*}}",
			[]node{
				&sectionNode{"*", mustPath("*"), false, []node{
					textNode("("),
					&varNode{".", mustPath("."), htmlEscape},
					textNode(")"),
				}},
			},
		},
		{
			"{{#list}}({{*}}){{/list}}",
			[]node{
				&sectionNode{"list", mustPath("list"), false, []node{
					textNode("("),
					&varNode{"*", mustPath("*"), htmlEscape},
					textNode(")"),
				}},
			},
		},
		{
			"{{#test_value {{foo}} \"bar\"}}({{a}a}}){{/test_value}}",
			[]node{
				&testNode{mustPath("foo"), "bar", []node{
					textNode("("),
					&varNode{"a}a", mustPath("a}a"), htmlEscape},
					textNode(")"),
				}},
			},
		},
		{
			"{{#test_value {{foo}} \"bar\"}}{{#a}}{{b}}{{/a}}{{/test_value}}",
			[]node{
				&testNode{mustPath("foo"), "bar", []node{
					&sectionNode{"a", mustPath("a"), false, []node{
						&varNode{"b", mustPath("b"), htmlEscape},
					}},
				}},
			},
		},
		{
			"{{#list}}({{a}a}}){{/list}}",
			[]node{
				&sectionNode{"list", mustPath("list"), false, []node{
					textNode("("),
					&varNode{"a}a", mustPath("a}a"), htmlEscape},
					textNode(")"),
				}},
			},
		},
		{
			"{{~customize}}blah blah{{/customize}}",
			[]node{
				&functionSectionNode{
					"customize",
					nil,
					[]node{
						textNode("blah blah"),
					},
				},
			},
		},
		{ // so that we can do something like {{~lowercase locale="EN-US"}}...
			`{{~customize opt1="value1" opt2="value2"}}blah blah{{/customize}}`,
			[]node{
				&functionSectionNode{
					"customize",
					map[string]string{"opt1": "value1", "opt2": "value2"},
					[]node{
						textNode("blah blah"),
					},
				},
			},
		},
		{
			`{{ metrics."http.request.count" }}`,
			[]node{
				&varNode{`metrics."http.request.count"`, mustPath(`metrics."http.request.count"`), htmlEscape},
			},
		},
		{
			`{{ fields.'service.name'.value }}`,
			[]node{
				&varNode{`fields.'service.name'.value`, mustPath(`fields.'service.name'.value`), htmlEscape},
			},
		},
		{
			`{{#config."feature.flags"}}{{enabled}}{{/config."feature.flags"}}`,
			[]node{
				&sectionNode{`config."feature.flags"`, mustPath(`config."feature.flags"`), false, []node{
					&varNode{"enabled", mustPath("enabled"), htmlEscape},
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
			`failed to find closing tag for section "test_value"`,
		},
		{
			"{{#alert.severity.isCritical}}\n    test content\n    {{/alert.severity.isError}}",
			`failed to find closing tag for section "alert.severity.isCritical"`,
		},
		{
			"{{#foo}}hello{{/bar}}",
			`failed to find closing tag for section "foo"`,
		},
		{
			"{{#test_value a b}}",
			`1:14 syntax error: unexpected token t_error:"Missing test_value identifier"`,
		},
		{
			"{{#test_value {{a b\"}}",
			`1:22 syntax error: unexpected token t_error:"invalid test_value value token"`,
		},
		{
			`{{ "foo }}`,
			`unterminated quote`,
		},
		{
			`{{ x."a\q" }}`,
			`invalid escape`,
		},
		{
			`{{{ x."a }}}`,
			`unterminated quote`,
		},
		{
			`{{ x."a"b }}`,
			`unexpected`,
		},
		{
			`{{#a."b"}}x{{/a.'b'}}`,
			`failed to find closing tag`,
		},
	} {
		parser := newParser(newLexer(test.template, "{{", "}}", true), htmlEscape)
		_, err := parser.parse()
		if err == nil || !strings.Contains(err.Error(), test.expErr) {
			t.Errorf("expect error: %q, got %q", test.expErr, err)
		}
	}
}

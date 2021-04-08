// Copyright (c) 2014 Alex Kalyvitis

package mustache

import (
	"reflect"
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
				&varNode{"bar", true},
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
					&varNode{".", true},
					textNode(")"),
				}},
			},
		},
		{
			"{{#*}}({{.}}){{/*}}",
			[]node{
				&sectionNode{"*", false, []node{
					textNode("("),
					&varNode{".", true},
					textNode(")"),
				}},
			},
		},
		{
			"{{#list}}({{*}}){{/list}}",
			[]node{
				&sectionNode{"list", false, []node{
					textNode("("),
					&varNode{"*", true},
					textNode(")"),
				}},
			},
		},
		{
			"{{#list}}({{a}a}}){{/list}}",
			[]node{
				&sectionNode{"list", false, []node{
					textNode("("),
					&varNode{"a}a", true},
					textNode(")"),
				}},
			},
		},
	} {
		parser := newParser(newLexer(test.template, "{{", "}}"))
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

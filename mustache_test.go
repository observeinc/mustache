// Copyright (c) 2014 Alex Kalyvitis

package mustache

import (
	"bytes"
	"strings"

	"testing"
)

func TestTemplate(t *testing.T) {
	input := strings.NewReader("some text {{foo}} here")
	template := New()
	err := template.Parse(input)
	if err != nil {
		t.Error(err)
	}
	var output bytes.Buffer
	err = template.Render(&output, map[string]string{"foo": "bar %2B"})
	if err != nil {
		t.Error(err)
	}
	expected := "some text bar %2B here"
	if output.String() != expected {
		t.Errorf("expected %q got %q", expected, output.String())
	}
}

func TestFalsyTemplate(t *testing.T) {
	input := strings.NewReader("some text {{^foo}}{{foo}}{{/foo}} {{bar}} here")
	template := New()
	err := template.Parse(input)
	if err != nil {
		t.Error(err)
	}
	var output bytes.Buffer
	err = template.Render(&output, map[string]interface{}{"foo": 0, "bar": false})
	if err != nil {
		t.Error(err)
	}
	expected := "some text 0 false here"
	if output.String() != expected {
		t.Errorf("expected %q got %q", expected, output.String())
	}
}

func TestParseTree(t *testing.T) {
	template := New()
	template.elems = []node{
		textNode("Lorem ipsum dolor sit "),
		&varNode{"foo", mustPath("foo"), noEscape},
		textNode(", "),
		&sectionNode{"bar", mustPath("bar"), false, []node{
			&varNode{"baz", mustPath("baz"), htmlEscape},
			textNode(" adipiscing"),
		}},
		textNode(" elit. Proin commodo viverra elit "),
		&varNode{"zer", mustPath("zer"), noEscape},
		textNode("."),
	}
	data := map[string]interface{}{
		"foo": "amet",
		"bar": map[string]string{"baz": "consectetur"},
		"zer": 0.11,
	}
	b := bytes.NewBuffer(nil)
	w := newWriter(b)
	for _, e := range template.elems {
		err := e.render(template, w, data)
		if err != nil {
			t.Error(err)
		}
	}
	w.flush()

	expected := `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Proin commodo viverra elit 0.11.`

	if expected != b.String() {
		t.Errorf("output didn't match. expected %q got %q.", expected, b.String())
		t.Log(b.String())
	}
}

func TestTemplateJsonEscaped(t *testing.T) {
	input := strings.NewReader("some text {{foo}} here")
	template := New(JsonEscape())
	err := template.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = template.Render(&output, map[string]string{"foo": "\"bar\"\n<baz> %2B"})
	if err != nil {
		t.Fatal(err)
	}
	expected := "some text \\\"bar\\\"\\n<baz> %2B here"
	if output.String() != expected {
		t.Errorf("expected %q got %q", expected, output.String())
	}
}
func TestObjectOutput(t *testing.T) {
	inputTemplate := strings.NewReader("Raw output here: {{.}}")
	inputData := map[string]map[string]string{"foo": {"bar": "baz"}}
	template := New()
	err := template.Parse(inputTemplate)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = template.Render(&output, inputData)
	if err != nil {
		t.Fatal(err)
	}
	expected := "Raw output here: {&quot;foo&quot;:{&quot;bar&quot;:&quot;baz&quot;}}"
	if output.String() != expected {
		t.Errorf("expected %q got %q", expected, output.String())
	}
}

func TestObjectOutputJsonEscaped(t *testing.T) {
	inputTemplate := strings.NewReader("Raw output here: {{.}}")
	//inputData := map[string]map[string]string{"foo": {"bår": "baz"}}
	inputData := map[string]map[string]string{"foo": {"bar": "baz %2B"}}
	template := New(JsonEscape())
	err := template.Parse(inputTemplate)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = template.Render(&output, inputData)
	if err != nil {
		t.Fatal(err)
	}
	expected := "Raw output here: {\\\"foo\\\":{\\\"bar\\\":\\\"baz %2B\\\"}}"
	if output.String() != expected {
		t.Errorf("expected %q got %q", expected, output.String())
	}
}

func TestObjectOutputUnescaped(t *testing.T) {
	inputTemplate := strings.NewReader("Raw output here: {{{.}}}")
	inputData := map[string]map[string]string{"foo": {"bar": "baz %2B"}}
	template := New()
	err := template.Parse(inputTemplate)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = template.Render(&output, inputData)
	if err != nil {
		t.Fatal(err)
	}
	expected := "Raw output here: {\"foo\":{\"bar\":\"baz %2B\"}}"
	if output.String() != expected {
		t.Errorf("expected %q got %q", expected, output.String())
	}
}

func TestInterpolationWithWhitespace(t *testing.T) {
	input := strings.NewReader("some text {{foo.bar baz.foo}} here")
	template := New()
	err := template.Parse(input)
	if err != nil {
		t.Error(err)
	}
	var output bytes.Buffer
	err = template.Render(&output, map[string]map[string]map[string]string{"foo": {"bar baz": {"foo": "bar %2B"}}})
	if err != nil {
		t.Error(err)
	}
	expected := "some text bar %2B here"
	if output.String() != expected {
		t.Errorf("expected %q got %q", expected, output.String())
	}
}

func TestCustomFunctions(t *testing.T) {
	input := strings.NewReader(`raw text {{~reverse}}txet erom{{/reverse}}`)
	template := New(
		CustomizeFunction("reverse", func(s string) (string, error) {
			orig := []rune(s)
			l := len(orig)
			reversed := make([]rune, len(orig))
			for i, j := 0, l-1; i < len(orig); i, j = i+1, j-1 {
				reversed[j] = orig[i]
			}
			return string(reversed), nil
		}),
	)

	err := template.Parse(input)
	if err != nil {
		t.Error(err)
	}
	var output bytes.Buffer
	err = template.Render(&output, nil)
	if err != nil {
		t.Error(err)
	}
	expected := `raw text more text`
	if output.String() != expected {
		t.Errorf("expected %q got %q", expected, output.String())
	}
}

func TestCustomFunctionsWithOptions(t *testing.T) {
	input := strings.NewReader(`split: {{~split token="x"}}hexllxoxworxld{{/split}}`)
	template := New(
		CustomizeFunctionWithOptions("split", func(s string, opts map[string]string) (string, error) {
			return strings.Join(strings.Split(s, opts["token"]), " "), nil
		}),
	)

	err := template.Parse(input)
	if err != nil {
		t.Error(err)
	}
	var output bytes.Buffer
	err = template.Render(&output, nil)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v", output)
	expected := `split: he ll o wor ld`
	if output.String() != expected {
		t.Errorf("expected %q got %q", expected, output.String())
	}
}

type templateTest struct {
	template string
	payload  interface{}
	expect   string
}

func TestSectionTestValue(t *testing.T) {
	tests := []templateTest{
		{ // test basic
			`some text {{#test_value {{a}} "value"}}hidden{{/test_value}} here`,
			map[string]string{"a": "value"},
			`some text hidden here`,
		},
		{ // Expect top level context.
			`some text {{#test_value {{a}} "value"}}{{b}}{{/test_value}} here`,
			map[string]string{"a": "value", "b": "hidden"},
			`some text hidden here`,
		},
		{ // Test nesting.
			`some text {{#test_value {{a}} "value"}}{{#test_value {{b}} "hidden"}}thing{{/test_value}}{{/test_value}} here`,
			map[string]string{"a": "value", "b": "hidden"},
			`some text thing here`,
		},
		{ // test nil lookup
			`some text {{#test_value {{aa}} "value"}}hidden{{/test_value}} here`,
			map[string]string{"a": "value"},
			`some text  here`,
		},
		{ // Test nesting normal section
			`some text {{#test_value {{a}} "value"}}{{#b}}{{.}}{{/b}}{{/test_value}} here`,
			map[string]interface{}{"a": "value", "b": []int{1, 2, 3}},
			`some text 123 here`,
		},
	}
	for _, test := range tests {
		input := strings.NewReader(test.template)
		template := New(TestValueSection())
		err := template.Parse(input)
		if err != nil {
			t.Error(err)
		}
		var output bytes.Buffer
		err = template.Render(&output, test.payload)
		if err != nil {
			t.Error(err)
		}
		if output.String() != test.expect {
			t.Errorf("expected %q got %q", test.expect, output.String())
		}

	}

}

func TestPartialsCannotCycle(t *testing.T) {
	innerTemplate := New(Name("inner"))
	err := innerTemplate.Parse(strings.NewReader(`I am the inner.{{>outer}}`))
	if err != nil {
		t.Error(err)
	}

	outerTemplate := New(Name("outer"))
	err = outerTemplate.Parse(strings.NewReader(`I am the outer.{{>inner}}`))
	if err != nil {
		t.Error(err)
	}

	mainTemplate := New(Partial(outerTemplate), Partial(innerTemplate))
	err = mainTemplate.Parse(strings.NewReader(`{{>outer}}`))
	if err != nil {
		t.Error(err)
	}

	var output bytes.Buffer
	err = mainTemplate.Render(&output)
	if err != nil {
		t.Error(err)
	}

	expected := `I am the outer.I am the inner.`
	if output.String() != expected {
		t.Errorf("expected %q got %q", expected, output.String())
	}
}

func TestQuotedKeyTemplates(t *testing.T) {
	for _, test := range []templateTest{
		{ // double-quoted key containing dots
			`{{ metrics."http.request.count" }}`,
			map[string]interface{}{"metrics": map[string]interface{}{"http.request.count": 42}},
			`42`,
		},
		{ // single quotes are equivalent
			`{{ metrics.'http.request.count' }}`,
			map[string]interface{}{"metrics": map[string]interface{}{"http.request.count": 42}},
			`42`,
		},
		{ // non-terminal quoted segment, then a normal key
			`{{ a."b.c".d }}`,
			map[string]interface{}{"a": map[string]interface{}{"b.c": map[string]interface{}{"d": "X"}}},
			`X`,
		},
		{ // backslash-escaped quote inside a key
			`{{ files."a\".b" }}`,
			map[string]interface{}{"files": map[string]interface{}{`a".b`: "ok"}},
			`ok`,
		},
		{ // HTML escaping still applies to the resolved value
			`{{ x."a.b" }}`,
			map[string]interface{}{"x": map[string]interface{}{"a.b": "<v>"}},
			`&lt;v&gt;`,
		},
		{ // triple-brace leaves the value raw
			`{{{ x."a.b" }}}`,
			map[string]interface{}{"x": map[string]interface{}{"a.b": "<v>"}},
			`<v>`,
		},
		{ // empty quoted key addresses the empty-string key
			`{{ m."" }}`,
			map[string]interface{}{"m": map[string]interface{}{"": "E"}},
			`E`,
		},
		{ // a quoted "." is a literal key, not the whole-context shortcut
			`{{ obj."." }}`,
			map[string]interface{}{"obj": map[string]interface{}{".": "dot"}},
			`dot`,
		},
		{ // a section whose name is a quoted key, iterating a slice
			`{{#groups."a.b"}}{{name}} {{/groups."a.b"}}`,
			map[string]interface{}{"groups": map[string]interface{}{"a.b": []interface{}{
				map[string]interface{}{"name": "x"},
				map[string]interface{}{"name": "y"},
			}}},
			`x y `,
		},
	} {
		template := New(SilentMiss(false))
		if err := template.ParseString(test.template); err != nil {
			t.Errorf("parse %q: %v", test.template, err)
			continue
		}
		out, err := template.RenderString(test.payload)
		if err != nil {
			t.Errorf("render %q: %v", test.template, err)
			continue
		}
		if out != test.expect {
			t.Errorf("template %q: expected %q got %q", test.template, test.expect, out)
		}
	}
}

func TestRenderJSONBodyWithQuotedKeys(t *testing.T) {
	// A realistic use: build a JSON payload whose values come from a map keyed by
	// dotted field names, each holding a nested "value". It mixes quoted (dotted)
	// and unquoted keys; SilentMiss(false) makes any missed lookup fail the test.
	body := `{"service":"{{ fields."service.name".value }}",` +
		`"env":"{{ fields."deployment.environment.name".value }}",` +
		`"status":{{ fields."http.response.status_code".value }},` +
		`"seq":{{ fields.seqNum.value }},"ok":true}`

	template := New(SilentMiss(false))
	if err := template.ParseString(body); err != nil {
		t.Fatalf("parse: %v", err)
	}

	context := map[string]interface{}{
		"fields": map[string]interface{}{
			"service.name":                map[string]interface{}{"value": "checkout"},
			"deployment.environment.name": map[string]interface{}{"value": "production"},
			"http.response.status_code":   map[string]interface{}{"value": 500},
			"seqNum":                      map[string]interface{}{"value": 7},
		},
	}

	got, err := template.RenderString(context)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := `{"service":"checkout","env":"production","status":500,"seq":7,"ok":true}`
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

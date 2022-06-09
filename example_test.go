// Copyright (c) 2014 Alex Kalyvitis

package mustache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
)

func ExampleTemplate_basic() {
	template := New()
	parseErr := template.ParseString(`{{#foo}}{{bar}}{{/foo}}`)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "failed to parse template: %s\n", parseErr)
	}

	context := map[string]interface{}{
		"foo": true,
		"bar": "bazinga!",
	}

	output, err := template.RenderString(context)
	fmt.Println(output)
	// Output: bazinga!
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to render template: %s\n", err)
	}
}

func ExampleTemplate_partials() {
	partial := New(Name("partial"))
	parseErr := partial.ParseString(`{{bar}}`)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "failed to parse template: %s\n", parseErr)
	}

	template := New(Partial(partial))
	templateErr := template.ParseString(`{{#foo}}{{>partial}}{{/foo}}`)
	if templateErr != nil {
		fmt.Fprintf(os.Stderr, "failed to parse template: %s\n", templateErr)
	}

	context := map[string]interface{}{
		"foo": true,
		"bar": "bazinga!",
	}

	err := template.Render(os.Stdout, context)
	// Output: bazinga!

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to render template: %s\n", err)
	}
}

func ExampleTemplate_reader() {
	f, err := os.Open("template.mustache")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open file: %s\n", err)
	}
	t, err := Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse template: %s\n", err)
	}
	err = t.Render(os.Stdout, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to render template: %s\n", err)
	}

}

func ExampleTemplate_http() {
	writer := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "http://example.com?foo=bar&bar=one&bar=two", nil)

	template := New()
	err := template.ParseString(`
<ul>{{#foo}}<li>{{.}}</li>{{/foo}}</ul>
<ul>{{#bar}}<li>{{.}}</li>{{/bar}}</ul>`)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse template: %s\n", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		err := template.Render(w, r.URL.Query())
		if err != nil {
			fmt.Fprint(w, err.Error())
		}
	}

	handler(writer, request)

	fmt.Println(writer.Body.String())
	// Output:
	// <ul><li>bar</li></ul>
	// <ul><li>one</li><li>two</li></ul>
}

func ExampleOption() {
	title := New(Name("header"))               // instantiate and name the template
	titleErr := title.ParseString("{{title}}") // parse a template string
	// If there was an error do something with it.
	if titleErr != nil {
		fmt.Fprintf(os.Stderr, "failed to parse template: %s\n", titleErr)
	}

	body := New()
	body.Option(Name("body")) // options can be defined after we instantiate too
	parseErr := body.ParseString("{{content}}")
	// If there was an error do something with it.
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "failed to parse template: %s\n", parseErr)
	}

	template := New(
		Delimiters("|", "|"), // set the mustache delimiters to | instead of {{
		SilentMiss(false),    // return an error if a variable lookup fails
		Partial(title),       // register a partial
		Partial(body))        // and another one...

	templateErr := template.ParseString("|>header|\n|>body|")
	if templateErr != nil {
		fmt.Fprintf(os.Stderr, "failed to parse template: %s\n", templateErr)
	}

	context := map[string]interface{}{
		"title":   "Mustache",
		"content": "Logic less templates with Mustache!",
	}

	renderErr := template.Render(os.Stdout, context)

	// Output: Mustache
	// Logic less templates with Mustache!

	// If there was an error do something with it.
	if renderErr != nil {
		fmt.Fprintf(os.Stderr, "failed to render template: %s\n", renderErr)
	}
}

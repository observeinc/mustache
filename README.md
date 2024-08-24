# Mustache

[![godoc reference](https://godoc.org/github.com/observeinc/mustache?status.svg)](https://godoc.org/github.com/observeinc/mustache)
[![Tests Actions Status](https://github.com/observeinc/mustache/workflows/Go/badge.svg)](https://github.com/observeinc/mustache/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/observeinc/mustache)](https://goreportcard.com/report/github.com/observeinc/mustache)

This is an implementation of the mustache templating language in Go.

It is forked from [alexkappa/mustache](https://github.com/alexkappa/mustache) who was inspired by by [hoisie/mustache](https://github.com/hoisie/mustache).

It is built using lexing techniques described in the slides on [lexical scanning in Go](https://talks.golang.org/2011/lex.slide), and functional options as described in the blog post on [self-referential functions and the design of options](http://commandcenter.blogspot.nl/2014/01/self-referential-functions-and-design.html).

This package aims to cover 100% of the mustache specification tests, however by the time of this writing it is not complete. (It is missing some pieces around partials, lambdas, and inheritance.) 

On the flip side it also has some alternate rules for escaping. Supports golang structs by field name, method, or tag. Has some extensions to the spec relating to array indexing, and a conditional value match. 

For more information on mustache check the [official documentation](http://mustache.github.io/) and the [mustache spec](http://github.com/mustache/spec).

# Installation

Install with `go get github.com/observeinc/mustache`.

# Documentation

The API documentation is available at [godoc.org](http://godoc.org/github.com/observeinc/mustache).

# Usage

The core of this package is the `Template`, and it's `Parse` and `Render` functions.

```Go
template := mustache.New()
template.Parse(strings.NewReader("Hello, {{subject}}!"))
template.Render(os.Stdout, map[string]string{"subject": "world"})
```
## Helpers

There are additional `Parse` and `Render` helpers to deal with different kind of input or output, such as `string`, `[]byte` or `io.Writer`/`io.Reader`.

```Go
Parse(r io.Reader) error
ParseString(s string) error
ParseBytes(b []byte) error
```

```Go
Render(w io.Writer, context interface{}) error
RenderString(context interface{}) (string, error)
RenderBytes(context interface{}) ([]byte, error)
```

### Reader/Writer

```Go
f, err := os.Open("template.mustache")
if err != nil {
    fmt.Fprintf(os.Stderr, "failed to open file: %s\n", err)
}
t, err := Parse(f)
if err != nil {
    fmt.Fprintf(os.Stderr, "failed to parse template: %s\n", err)
}
t.Render(os.Stdout, nil)
```

**Note:** in the example above, we used [Parse](http://godoc.org/github.com/observeinc/mustache#Parse) which wraps the `t := New()` and `t.Parse()` functions for consiceness.

### String

```Go
t := mustache.New()
err := t.ParseString("Hello, {{subject}}!")
if err != nil {
    // handle error
}
s, _ := t.RenderString(map[string]string{"subject": "world"})
if err != nil {
    // handle error
}
fmt.Println(s)
```

## Options

It is possible to define some options on the template, which will alter the way the template will parse, render or fail.

The options are:

- `Name(n string) Option` sets the name of the template. This option is useful when using the template as a partial to another template.
- `Delimiters(start, end string) Option` sets the start and end delimiters of the template.
- `Partial(p *Template) Option` sets p as a partial to the template. It is important to set the name of p so that it may be looked up by the parent template.
- `SilentMiss(silent bool) Option` sets missing variable lookup behaviour.
- `HtmlEscape() Option` and `JsonEscape() Option` set the escaping mode for when tokens are substituted. The default is `HtmlEscape` which is what is specified by the mustache spec. `JsonEscape` will instead use escapes as needed for JSON encoding.

Options can be defined either as arguments to [New](http://godoc.org/github.com/observeinc/mustache#New) or using the [Option](http://godoc.org/github.com/observeinc/mustache#Template.Option) function.

## Partials

Partials are templates themselves and can be defined using the [Partial](http://godoc.org/github.com/observeinc/mustache#Partial) option.

**Note:** It is important to name the partial using the [Name](http://godoc.org/github.com/observeinc/mustache#Name) option which should match the mustache partial tag `{{>name}}` in the parent template.

```Go
title := New(
    Name("header")        // instantiate and name the template
    Delimiters("|", "|")) // set the mustache delimiters to | instead of {{

title.ParseString("|title|") // parse a template string

body := New()
body.Option(Name("body"))
body.ParseString("{{content}}")

template := New(
    SilentMiss(false), // return an error if a variable lookup fails
    Partial(title),    // register a partial
    Partial(body))     // and another one...

template.ParseString("{{>header}}\n{{>body}}")

context := map[string]interface{}{
    "title":   "Mustache",
    "content": "Logic less templates with Mustache!",
}

template.Render(os.Stdout, context)
```

## Functions

**note:** This is an extension to the mustache spec and library added by Observe Inc.

Functions are additional functionality that a template can invoke to mutate a rendered section. The use caes is to allow escaping, encoding, alterations, or other manipulations of rendered text without applying it to the whole document. The impetus was the need to allow a URL to be properly formatted and encoded in a JSON document. The JSON encoding for the whole document applies, but the specific value must be treated special first. As another example, a custom function named `lowercase` could be added that would be invoked like:

```mustache
Your name in lowercase letters is:{{~lowercase}}{{name.first}} {{name.last}}{{/lowercase}}
```

Installing this customizing function is similar to partials, where you use an `Option` in the creation.

```go
func tolower(in string) (string, error) {
    // TODO: implement a lowercase operation
    return "", nil
}

tmpl := New(
    CustomizeFunction("lowercase", tolower),
)
```

# Tests

Run `go test` as usual. If you want to run the spec tests against this package, make sure you've checked out the specs submodule. Otherwise spec tests will be skipped.

Currently certain spec tests are skipped as they fail due to an issue with how standalone tags and empty lines are being handled. Inspecting them manually, one can see that the templates render correctly but with some additional `\n` which should have been omited. See issue [#1](http://github.com/alexkappa/mustache/issues/1).

See [SPEC.md](https://github.com/observeinc/mustache/blob/master/SPEC.md) for a breakdown of which spec tests pass and fail.

# Contributing

If you would like to contribute, head on to the [issues](https://github.com/observeinc/mustache/issues) page for tasks that need help.

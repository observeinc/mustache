package mustache

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestSilentMissSurfacesCustomizerErrors(t *testing.T) {
	fail := CustomizeFunction("fail", func(s string) (string, error) {
		return "", errors.New("customizer boom")
	})
	pass := CustomizeFunction("pass", func(s string) (string, error) {
		return strings.ToUpper(s), nil
	})

	t.Run("top_level_customizer_error", func(t *testing.T) {
		tmpl := New(SilentMiss(true), fail)
		if err := tmpl.ParseString(`{{~fail}}body{{/fail}}`); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if _, err := tmpl.RenderString(nil); err == nil {
			t.Fatal("expected customizer error, got nil")
		}
	})

	t.Run("missing_variable_still_silent", func(t *testing.T) {
		tmpl := New(SilentMiss(true), pass)
		if err := tmpl.ParseString(`hello {{name}}`); err != nil {
			t.Fatalf("parse: %v", err)
		}
		got, err := tmpl.RenderString(nil)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if got != "hello " {
			t.Fatalf("got %q want %q", got, "hello ")
		}
	})

	t.Run("inside_section", func(t *testing.T) {
		tmpl := New(SilentMiss(true), fail)
		if err := tmpl.ParseString(`{{#items}}{{~fail}}x{{/fail}}{{/items}}`); err != nil {
			t.Fatalf("parse: %v", err)
		}
		_, err := tmpl.RenderString(map[string]interface{}{
			"items": []interface{}{map[string]string{"k": "v"}},
		})
		if err == nil {
			t.Fatal("expected customizer error inside section, got nil")
		}
	})

	t.Run("inside_function_section", func(t *testing.T) {
		tmpl := New(SilentMiss(true), fail, pass)
		if err := tmpl.ParseString(`{{~pass}}{{~fail}}x{{/fail}}{{/pass}}`); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if _, err := tmpl.RenderString(nil); err == nil {
			t.Fatal("expected nested customizer error, got nil")
		}
	})

	t.Run("inside_partial", func(t *testing.T) {
		partial := New(Name("p"), SilentMiss(true), fail)
		if err := partial.ParseString(`{{~fail}}x{{/fail}}`); err != nil {
			t.Fatalf("parse partial: %v", err)
		}
		tmpl := New(SilentMiss(true), Partial(partial))
		if err := tmpl.ParseString(`{{>p}}`); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if _, err := tmpl.RenderString(nil); err == nil {
			t.Fatal("expected customizer error from partial, got nil")
		}
	})
}

func TestSilentMissOffMissingVariableErrors(t *testing.T) {
	tmpl := New(SilentMiss(false))
	if err := tmpl.ParseString(`hello {{name}}`); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := tmpl.RenderString(nil); err == nil {
		t.Fatal("expected missing-variable error with SilentMiss(false)")
	}
}

func TestFunctionSectionStandaloneLine(t *testing.T) {
	upper := CustomizeFunction("upper", func(s string) (string, error) {
		return strings.ToUpper(s), nil
	})
	empty := CustomizeFunction("empty", func(s string) (string, error) {
		return "", nil
	})

	t.Run("non_empty_survives", func(t *testing.T) {
		tmpl := New(SilentMiss(false), upper)
		if err := tmpl.ParseString("before\n{{~upper}}x{{/upper}}\nafter"); err != nil {
			t.Fatalf("parse: %v", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Render(&buf, nil); err != nil {
			t.Fatalf("render: %v", err)
		}
		want := "before\nX\nafter"
		if buf.String() != want {
			t.Fatalf("got %q want %q", buf.String(), want)
		}
	})

	t.Run("empty_body_emits_nothing", func(t *testing.T) {
		tmpl := New(SilentMiss(false), empty)
		if err := tmpl.ParseString("before\n{{~empty}}{{/empty}}\nafter"); err != nil {
			t.Fatalf("parse: %v", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Render(&buf, nil); err != nil {
			t.Fatalf("render: %v", err)
		}
		// Empty customizer output returns early; the standalone tag line is
		// still subject to the writer's tag-only-line elision.
		want := "before\nafter"
		if buf.String() != want {
			t.Fatalf("got %q want %q", buf.String(), want)
		}
	})
}

func TestFunctionOptionEscapes(t *testing.T) {
	var seen map[string]string
	opt := CustomizeFunctionWithOptions("echo", func(s string, opts map[string]string) (string, error) {
		seen = opts
		return opts["sep"] + s + opts["sep"], nil
	})

	t.Run("escaped_quote", func(t *testing.T) {
		seen = nil
		tmpl := New(opt)
		if err := tmpl.ParseString(`{{~echo sep="\""}}x{{/echo}}`); err != nil {
			t.Fatalf("parse: %v", err)
		}
		got, err := tmpl.RenderString(nil)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if seen["sep"] != `"` {
			t.Fatalf("opts[sep]=%q want %q", seen["sep"], `"`)
		}
		if got != `"x"` {
			t.Fatalf("got %q want %q", got, `"x"`)
		}
	})

	t.Run("escaped_backslash", func(t *testing.T) {
		seen = nil
		tmpl := New(opt)
		if err := tmpl.ParseString(`{{~echo sep="a\\b"}}x{{/echo}}`); err != nil {
			t.Fatalf("parse: %v", err)
		}
		got, err := tmpl.RenderString(nil)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if seen["sep"] != `a\b` {
			t.Fatalf("opts[sep]=%q want %q", seen["sep"], `a\b`)
		}
		if got != `a\bxa\b` {
			t.Fatalf("got %q want %q", got, `a\bxa\b`)
		}
	})

	t.Run("unescaped_still_works", func(t *testing.T) {
		seen = nil
		tmpl := New(opt)
		// Leading text keeps this case independent of the standalone-line
		// writer fix; it only asserts option parsing.
		if err := tmpl.ParseString(`wrap:{{~echo sep="-"}}x{{/echo}}`); err != nil {
			t.Fatalf("parse: %v", err)
		}
		got, err := tmpl.RenderString(nil)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if seen["sep"] != "-" {
			t.Fatalf("opts[sep]=%q want %q", seen["sep"], "-")
		}
		if got != "wrap:-x-" {
			t.Fatalf("got %q want %q", got, "wrap:-x-")
		}
	})
}

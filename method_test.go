// Copyright (c) 2014 Alex Kalyvitis

package mustache

import "testing"

type methodContext struct {
	First string
}

// Greeting is a zero-argument method returning a single value — the case that
// should resolve as a template variable.
func (m methodContext) Greeting() string { return "hello" }

// Nothing has no return value; it must never be invoked as a value.
func (m methodContext) Nothing() {}

// Echo requires an argument; it must never be called with zero arguments.
func (m methodContext) Echo(s string) string { return s }

// TestStructMethodLookup verifies that a zero-argument method on the context is
// resolved and its return value rendered. This guards against the receiver-bound
// method having NumIn()==0 (not 1), which previously caused the method branch to
// be skipped entirely.
func TestStructMethodLookup(t *testing.T) {
	tmpl := New(SilentMiss(false))
	if err := tmpl.ParseString(`{{First}}, {{Greeting}}`); err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := tmpl.RenderString(methodContext{First: "hi"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "hi, hello"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// TestStructMethodNoReturnIsSkipped verifies a method with no return value is not
// treated as a variable and does not panic on the missing first result.
func TestStructMethodNoReturnIsSkipped(t *testing.T) {
	tmpl := New() // SilentMiss default: a miss renders empty
	if err := tmpl.ParseString(`[{{Nothing}}]`); err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := tmpl.RenderString(methodContext{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "[]"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// TestStructMethodWithArgIsSkipped verifies a method requiring an argument is not
// called with zero arguments (which previously panicked) and simply does not
// resolve.
func TestStructMethodWithArgIsSkipped(t *testing.T) {
	tmpl := New()
	if err := tmpl.ParseString(`[{{Echo}}]`); err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := tmpl.RenderString(methodContext{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "[]"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

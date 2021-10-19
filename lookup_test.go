// Copyright (c) 2014 Alex Kalyvitis

package mustache

import (
	"reflect"
	"testing"
)

func TestSimpleLookup(t *testing.T) {
	var nilStrPtr *string
	stringForPtr := "string"
	for _, test := range []struct {
		context    interface{}
		assertions []struct {
			name  string
			value interface{}
			truth bool
		}
	}{
		{
			context: map[string]interface{}{
				"integer": 123,
				"string":  "abc",
				"boolean": true,
				"map": map[string]interface{}{
					"in": "I'm nested!",
				},
			},
			assertions: []struct {
				name  string
				value interface{}
				truth bool
			}{
				{"integer", 123, true},
				{"string", "abc", true},
				{"boolean", true, true},
				{"map.in", "I'm nested!", true},
			},
		},
		{
			context: struct {
				Integer  int
				String   string
				Boolean  bool
				Nested   struct{ Inside string }
				Tagged   string `mustache:"newName"`
				badTag   string `mustache:"fail"`
				bad      string
				ValidPtr *string
				NilPtr   *string
			}{
				123, "abc", true, struct{ Inside string }{"I'm nested!"}, "xyz", "bad", "bad", &stringForPtr, nil,
			},
			assertions: []struct {
				name  string
				value interface{}
				truth bool
			}{
				{"Integer", 123, true},
				{"String", "abc", true},
				{"Boolean", true, true},
				{"Nested.Inside", "I'm nested!", true},
				{"newName", "xyz", true},
				{"Tagged", "xyz", true},
				{"bad", nil, false},
				{"badTag", nil, false},
				{"fail", nil, false},
				{"ValidPtr", &stringForPtr, true},
				{"NilPtr", nilStrPtr, false},
			},
		},
	} {
		for _, assertion := range test.assertions {
			value, truth := lookup(assertion.name, test.context)
			if value != assertion.value {
				t.Errorf("Unexpected value %v,%T != %v,%T", value, value, assertion.value, assertion.value)
			}
			if truth != assertion.truth {
				t.Errorf("Unexpected truth %t != %t", truth, assertion.truth)
			}
		}
	}
}

func TestTruth(t *testing.T) {
	for _, test := range []struct {
		input    interface{}
		expected bool
	}{
		{"abc", true},
		{"", false},
		{123, true},
		{0, false},
		{true, true},
		{false, false},
	} {
		truth := truth(reflect.ValueOf(test.input))
		if truth != test.expected {
			t.Errorf("Unexpected truth %t != %t", truth, test.expected)
		}
	}
}

// Copyright (c) 2014 Alex Kalyvitis
// Portions Copyright (c) 2009 Michael Hoisie

package mustache

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// pathSegment is a single key within a dotted lookup path. quoted reports whether
// the segment originated from a quoted key (e.g. "deployment.environment.name"). For
// quoted segments the interior dots are literal and the "." whole-context shortcut
// does not apply, so a quoted "." addresses a real key named ".".
type pathSegment struct {
	key    string
	quoted bool
}

// parsePath splits a variable path into segments, honoring quoted key segments. A
// segment beginning with a double or single quote is read up to its
// matching closing quote; dots inside the quotes are literal and \\ and \" (or \')
// are the only recognized escapes. Quotes that are not at the start of a segment are
// ordinary characters. It returns an error for an unterminated quote, an invalid or
// dangling backslash escape, or unexpected characters following a closing quote.
//
// A lone "." is returned as a single unquoted segment so that lookup can resolve it
// to the whole current context (the {{.}} tag).
func parsePath(raw string) ([]pathSegment, error) {
	if raw == "." {
		return []pathSegment{{key: "."}}, nil
	}
	var segments []pathSegment
	i, n := 0, len(raw)
	for {
		var seg pathSegment
		if i < n && (raw[i] == '"' || raw[i] == '\'') {
			key, next, err := parseQuotedSegment(raw, i)
			if err != nil {
				return nil, err
			}
			seg = pathSegment{key: key, quoted: true}
			i = next
			if i < n && raw[i] != '.' {
				return nil, fmt.Errorf("unexpected %q after closing quote in path %q", string(raw[i]), raw)
			}
		} else {
			// Unquoted segment: everything up to the next dot is a literal key; any
			// quote characters here are ordinary runes.
			start := i
			for i < n && raw[i] != '.' {
				i++
			}
			seg = pathSegment{key: raw[start:i]}
		}
		segments = append(segments, seg)
		if i == n {
			break
		}
		i++ // consume the separator '.'
		if i == n {
			segments = append(segments, pathSegment{}) // trailing dot -> empty final segment
			break
		}
	}
	return segments, nil
}

// parseQuotedSegment reads a quoted key that begins at raw[start] (a quote rune) and
// returns the unescaped key together with the index just past the closing quote.
func parseQuotedSegment(raw string, start int) (string, int, error) {
	quote := raw[start]
	var b strings.Builder
	for i := start + 1; i < len(raw); i++ {
		switch c := raw[i]; c {
		case quote:
			return b.String(), i + 1, nil
		case '\\':
			if i+1 >= len(raw) {
				return "", 0, fmt.Errorf("dangling escape in path %q", raw)
			}
			esc := raw[i+1]
			if esc != '\\' && esc != quote {
				return "", 0, fmt.Errorf("invalid escape \\%s in path %q", string(esc), raw)
			}
			b.WriteByte(esc)
			i++ // skip the escaped character
		default:
			b.WriteByte(c)
		}
	}
	return "", 0, fmt.Errorf("unterminated quote in path %q", raw)
}

// lookupPath resolves a pre-parsed path against the context chain. The first segment
// is searched across the whole chain (most-specific context first); each subsequent
// segment drills into the value found by the previous one. A falsy intermediate value
// stops the walk (matching dotted-lookup behavior), but the final segment's value and
// truth are returned as-is, so a found-but-falsy value (a nil pointer, 0, false, "")
// is still returned.
func lookupPath(path []pathSegment, context ...interface{}) (interface{}, bool) {
	if len(path) == 0 {
		return nil, false
	}
	value, ok := resolveSegment(path[0], context...)
	for _, seg := range path[1:] {
		if !ok {
			return nil, false
		}
		value, ok = resolveSegment(seg, value)
	}
	return value, ok
}

// resolveSegment searches the context chain for a single path segment. It mirrors the
// per-context resolution used for dotted names: maps, structs, and arrays/slices are
// probed in turn, and an unquoted "." returns the whole context as-is.
func resolveSegment(seg pathSegment, context ...interface{}) (interface{}, bool) {
	for _, c := range context {
		reflectValue := reflect.ValueOf(c)
		// If the segment is an unquoted ".", return the whole context as-is. A quoted
		// "." is a literal key and falls through to the normal resolution below.
		if !seg.quoted && seg.key == "." {
			return c, truth(reflectValue)
		}
		switch reflectValue.Kind() {
		// If the current context is a map, we'll look for a key in that map
		// that matches the segment.
		case reflect.Map:
			val, ok, found := lookup_map(seg.key, reflectValue)
			if found {
				return val, ok
			}

		// If the current context is a struct, we'll look for a property in that
		// struct that matches the segment.
		case reflect.Struct:
			val, ok, found := lookup_struct(seg.key, reflectValue)
			if found {
				return val, ok
			}

		// If the current context is an array or slice, we'll try to find the segment
		// as an index in the context.
		case reflect.Array, reflect.Slice:
			val, ok, found := lookup_array(seg.key, reflectValue)
			if found {
				return val, ok
			}
		}
		// If by this point no value was matched, we'll move up a step in the
		// chain and try to match a value there.
	}
	// We've exhausted the whole context chain and found nothing. Return a nil
	// value and a negative truth.
	return nil, false
}

func lookup_map(name string, reflectValue reflect.Value) (value interface{}, ok bool, found bool) {
	item := reflectValue.MapIndex(reflect.ValueOf(name))
	if item.IsValid() {
		return item.Interface(), truth(item), true
	}
	return nil, false, false

}

func lookup_struct(name string, reflectValue reflect.Value) (value interface{}, ok bool, found bool) {
	field := reflectValue.FieldByName(name)
	if field.IsValid() && field.CanInterface() {
		return field.Interface(), truth(field), true
	}
	method := reflectValue.MethodByName(name)
	if method.IsValid() && method.Type().NumIn() == 1 {
		out := method.Call(nil)[0]
		return out.Interface(), truth(out), true
	}

	typ := reflectValue.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue
		}
		tag := f.Tag.Get("mustache")
		if tag == name {
			field := reflectValue.Field(i)
			if field.IsValid() {
				return field.Interface(), truth(field), true
			}
		}
	}
	return nil, false, false
}

func lookup_array(name string, reflectValue reflect.Value) (value interface{}, ok bool, found bool) {
	idx, err := strconv.Atoi(name)
	if err != nil {
		return nil, false, false
	}
	if reflectValue.Len() <= idx || idx < 0 {
		return nil, false, false
	}
	field := reflectValue.Index(idx)
	if field.IsValid() {
		return field.Interface(), truth(field), true
	}

	return nil, false, false
}

// The truth function will tell us if r is a truthy value or not. This is
// important for sections as they will render their content based on the output
// of this function.
//
// Zero values are considered falsy. For example an empty string, the integer 0
// and so on are all considered falsy.
func truth(r reflect.Value) bool {
out:
	switch r.Kind() {
	case reflect.Array, reflect.Slice:
		return r.Len() > 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return r.Int() > 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return r.Uint() > 0
	case reflect.Float32, reflect.Float64:
		return r.Float() > 0
	case reflect.String:
		return r.String() != ""
	case reflect.Bool:
		return r.Bool()
	case reflect.Ptr, reflect.Interface:
		r = r.Elem()
		goto out
	case reflect.Invalid:
		return false
	default:
		if r.CanInterface() {
			return r.Interface() != nil
		} else {
			return false
		}
	}
}

// Copyright (c) 2014 Alex Kalyvitis
// Portions Copyright (c) 2009 Michael Hoisie

package mustache

import (
	"reflect"
	"strconv"
	"strings"
)

// The lookup function searches for a property that matches name within the
// context chain. We first start from the first item in the context chain which
// is the most likely to have the value we're looking for. If not found, we'll
// move up the chain and repeat.
func lookup(name string, context ...interface{}) (interface{}, bool) {
	// If the dot notation was used we split the word in two and perform two
	// consecutive lookups. If the first one fails we return no value and a
	// negative truth. Taken from github.com/hoisie/mustache.
	if name != "." && strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)
		if value, ok := lookup(parts[0], context...); ok {
			return lookup(parts[1], value)
		}
		return nil, false
	}

	// Iterate over the context chain and try to match the name to a value.
	for _, c := range context {
		// Reflect on the value of the current context.
		reflectValue := reflect.ValueOf(c)
		// If the name is ".", we should return the whole context as-is.
		if name == "." {
			return c, truth(reflectValue)
		}
		switch reflectValue.Kind() {
		// If the current context is a map, we'll look for a key in that map
		// that matches the name.
		case reflect.Map:
			val, ok, found := lookup_map(name, reflectValue)
			if found {
				return val, ok
			}

		// If the current context is a struct, we'll look for a property in that
		// struct that matches the name.
		case reflect.Struct:
			val, ok, found := lookup_struct(name, reflectValue)
			if found {
				return val, ok
			}

		// If the current context is an array or slice, we'll try to find the current
		// name as an index in the context.
		case reflect.Array, reflect.Slice:
			val, ok, found := lookup_array(name, reflectValue)
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

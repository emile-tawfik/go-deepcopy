package deepcopy

import (
	"fmt"
	"reflect"
)

type TypeMap map[reflect.Type]func(interface{}) (interface{}, error)

type copier func(interface{}, map[uintptr]interface{}, TypeMap) (interface{}, error)

var copiers map[reflect.Kind]copier

func init() {
	copiers = map[reflect.Kind]copier{
		reflect.Bool:       _primitive,
		reflect.Int:        _primitive,
		reflect.Int8:       _primitive,
		reflect.Int16:      _primitive,
		reflect.Int32:      _primitive,
		reflect.Int64:      _primitive,
		reflect.Uint:       _primitive,
		reflect.Uint8:      _primitive,
		reflect.Uint16:     _primitive,
		reflect.Uint32:     _primitive,
		reflect.Uint64:     _primitive,
		reflect.Uintptr:    _primitive,
		reflect.Float32:    _primitive,
		reflect.Float64:    _primitive,
		reflect.Complex64:  _primitive,
		reflect.Complex128: _primitive,
		reflect.Array:      _array,
		reflect.Map:        _map,
		reflect.Ptr:        _pointer,
		reflect.Slice:      _slice,
		reflect.String:     _primitive,
		reflect.Struct:     _struct,
	}
}

// MustAnything does a deep copy and panics on any errors.
func MustAnything(x interface{}) interface{} {
	dc, err := Anything(x)
	if err != nil {
		panic(err)
	}
	return dc
}

// MustAnything does a deep copy and panics on any errors.
func MustAnythingWithCustomTypes(x interface{}, typeMap TypeMap) interface{} {
	dc, err := AnythingWithCustomTypes(x, typeMap)
	if err != nil {
		panic(err)
	}
	return dc
}

// Primitive makes a copy of a primitive type...which just means it returns the input value.
// This is wholly uninteresting, but I included it for consistency's sake.
func _primitive(x interface{}, ptrs map[uintptr]interface{}, _ TypeMap) (interface{}, error) {
	kind := reflect.ValueOf(x).Kind()
	if kind == reflect.Array ||
		kind == reflect.Chan ||
		kind == reflect.Func ||
		kind == reflect.Interface ||
		kind == reflect.Map ||
		kind == reflect.Ptr ||
		kind == reflect.Slice ||
		kind == reflect.Struct ||
		kind == reflect.UnsafePointer {
		return nil, fmt.Errorf("unable to copy %v (a %v) as a primitive", x, kind)
	}
	return x, nil
}

// Anything makes a deep copy of whatever gets passed in. It handles pretty much all known Go types
// (with the exception of channels, unsafe pointers, and functions). Note that this is a truly deep
// copy that will work it's way all the way to the leaves of the types--any pointer will be copied,
// any values in any slice or map will be deep copied, etc.
// Note: in order to avoid an infinite loop, we keep track of any pointers that we've run across.
// If we run into that pointer again, we don't make another deep copy of it; we just replace it with
// the copy we've already made. This also ensures that the cloned result is functionally equivalent
// to the original value.
func Anything(x interface{}) (interface{}, error) {
	ptrs := make(map[uintptr]interface{})
	return _anything(x, ptrs, nil)
}

func AnythingWithCustomTypes(x interface{}, typeMap TypeMap) (interface{}, error) {
	ptrs := make(map[uintptr]interface{})
	return _anything(x, ptrs, typeMap)
}

func _anything(x interface{}, ptrs map[uintptr]interface{}, typeMap TypeMap) (interface{}, error) {
	t := reflect.TypeOf(x)
	if len(typeMap) > 0 {
		if f, ok := typeMap[t]; ok {
			return f(x)
		}
	}
	v := reflect.ValueOf(x)
	if !v.IsValid() {
		return x, nil
	}
	if c, ok := copiers[v.Kind()]; ok {
		return c(x, ptrs, typeMap)
	}
	return nil, fmt.Errorf("unable to make a deep copy of %v (type: %v) - kind %v is not supported", x, t, v.Kind())
}

func _slice(x interface{}, ptrs map[uintptr]interface{}, typeMap TypeMap) (interface{}, error) {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("must pass a value with kind of Slice; got %v", v.Kind())
	}
	// Create a new slice and, for each item in the slice, make a deep copy of it.
	if v.IsNil() {
		return x, nil
	}
	size := v.Len()
	t := reflect.TypeOf(x)
	dc := reflect.MakeSlice(t, size, size)
	for i := 0; i < size; i++ {
		item, err := _anything(v.Index(i).Interface(), ptrs, typeMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone slice item at index %v: %v", i, err)
		}
		iv := reflect.ValueOf(item)
		if iv.IsValid() {
			dc.Index(i).Set(iv)
		}
	}
	return dc.Interface(), nil
}

func _map(x interface{}, ptrs map[uintptr]interface{}, typeMap TypeMap) (interface{}, error) {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Map {
		return nil, fmt.Errorf("must pass a value with kind of Map; got %v", v.Kind())
	}
	if v.IsNil() {
		return x, nil
	}
	t := reflect.TypeOf(x)
	dc := reflect.MakeMapWithSize(t, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		item, err := _anything(iter.Value().Interface(), ptrs, typeMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone map item %v: %v", iter.Key().Interface(), err)
		}
		k, err := _anything(iter.Key().Interface(), ptrs, typeMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone the map key %v: %v", k, err)
		}
		dc.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(item))
	}
	return dc.Interface(), nil
}

func _pointer(x interface{}, ptrs map[uintptr]interface{}, typeMap TypeMap) (interface{}, error) {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("must pass a value with kind of Ptr; got %v", v.Kind())
	}

	if v.IsNil() {
		t := reflect.TypeOf(x)
		return reflect.Zero(t).Interface(), nil
	}

	addr := v.Pointer()
	if dc, ok := ptrs[addr]; ok {
		return dc, nil
	}
	t := reflect.TypeOf(x)
	dc := reflect.New(t.Elem())
	ptrs[addr] = dc.Interface()

	item, err := _anything(v.Elem().Interface(), ptrs, typeMap)
	if err != nil {
		return nil, fmt.Errorf("failed to copy the value under the pointer %v: %v", v, err)
	}
	iv := reflect.ValueOf(item)
	if iv.IsValid() {
		dc.Elem().Set(reflect.ValueOf(item))
	}

	return dc.Interface(), nil
}

func _struct(x interface{}, ptrs map[uintptr]interface{}, typeMap TypeMap) (interface{}, error) {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("must pass a value with kind of Struct; got %v", v.Kind())
	}
	t := reflect.TypeOf(x)
	dc := reflect.New(t)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			return nil, fmt.Errorf("can't copy type %+v cause of field %v", t, f.Name)
		}
		item, err := _anything(v.Field(i).Interface(), ptrs, typeMap)
		if err != nil {
			return nil, fmt.Errorf("failed to copy the field %v in the struct %#v: %v", t.Field(i).Name, x, err)
		}
		dc.Elem().Field(i).Set(reflect.ValueOf(item))
	}
	return dc.Elem().Interface(), nil
}

func _array(x interface{}, ptrs map[uintptr]interface{}, typeMap TypeMap) (interface{}, error) {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Array {
		return nil, fmt.Errorf("must pass a value with kind of Array; got %v", v.Kind())
	}
	t := reflect.TypeOf(x)
	size := t.Len()
	dc := reflect.New(reflect.ArrayOf(size, t.Elem())).Elem()
	for i := 0; i < size; i++ {
		item, err := _anything(v.Index(i).Interface(), ptrs, typeMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone array item at index %v: %v", i, err)
		}
		dc.Index(i).Set(reflect.ValueOf(item))
	}
	return dc.Interface(), nil
}

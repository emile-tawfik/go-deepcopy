package deepcopy

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func ExampleAnything() {
	tests := []interface{}{
		`Now cut that out!`,
		39,
		true,
		false,
		2.14,
		[]string{
			"Phil Harris",
			"Rochester van Jones",
			"Mary Livingstone",
			"Dennis Day",
		},
		[2]string{
			"Jell-O",
			"Grape-Nuts",
		},
		[]int(nil),
		map[string]int(nil),
	}

	for _, expected := range tests {
		actual := MustAnything(expected)
		fmt.Printf("%#v\n", actual)
	}
	// Output:
	// "Now cut that out!"
	// 39
	// true
	// false
	// 2.14
	// []string{"Phil Harris", "Rochester van Jones", "Mary Livingstone", "Dennis Day"}
	// [2]string{"Jell-O", "Grape-Nuts"}
	// []int(nil)
	// map[string]int(nil)
}

func ExampleAnythingWithCustomTypes() {
	tests := []interface{}{
		`Now cut that out!`,
		39,
		true,
		false,
		2.14,
		[]string{
			"Phil Harris",
			"Rochester van Jones",
			"Mary Livingstone",
			"Dennis Day",
		},
		[2]string{
			"Jell-O",
			"Grape-Nuts",
		},
		[]int(nil),
		map[string]int(nil),
		time.Date(2021, time.November, 5, 21, 3, 35, 12, time.UTC),
	}

	typeMap := TypeMap{
		reflect.TypeOf(time.Time{}): func(i interface{}) (interface{}, error) {
			return i.(time.Time), nil
		},
	}

	for _, expected := range tests {
		actual := MustAnythingWithCustomTypes(expected, typeMap)
		fmt.Printf("%#v\n", actual)
	}
	// Output:
	// "Now cut that out!"
	// 39
	// true
	// false
	// 2.14
	// []string{"Phil Harris", "Rochester van Jones", "Mary Livingstone", "Dennis Day"}
	// [2]string{"Jell-O", "Grape-Nuts"}
	// []int(nil)
	// map[string]int(nil)
	// time.Date(2021, time.November, 5, 21, 3, 35, 12, time.UTC)
}

type Foo struct {
	Foo *Foo
	Bar int
}

func ExampleMap() {
	x := map[string]*Foo{
		"foo": &Foo{Bar: 1},
		"bar": &Foo{Bar: 2, Foo: &Foo{Bar: 3}},
	}
	y := MustAnything(x).(map[string]*Foo)
	for _, k := range []string{"foo", "bar"} { // to ensure consistent order
		fmt.Printf("x[\"%v\"] = y[\"%v\"]: %v\n", k, k, x[k] == y[k])
		fmt.Printf("x[\"%v\"].Foo = y[\"%v\"].Foo: %v\n", k, k, x[k].Foo == y[k].Foo)
		fmt.Printf("x[\"%v\"].Bar = y[\"%v\"].Bar: %v\n", k, k, x[k].Bar == y[k].Bar)
		if x[k].Foo != nil {
			fmt.Printf("x[\"%v\"].Foo.Bar = y[\"%v\"].Foo.Bar: %v\n", k, k, x[k].Foo.Bar == y[k].Foo.Bar)
		}
	}
	// Output:
	// x["foo"] = y["foo"]: false
	// x["foo"].Foo = y["foo"].Foo: true
	// x["foo"].Bar = y["foo"].Bar: true
	// x["bar"] = y["bar"]: false
	// x["bar"].Foo = y["bar"].Foo: false
	// x["bar"].Bar = y["bar"].Bar: true
	// x["bar"].Foo.Bar = y["bar"].Foo.Bar: true
}

func TestInterface(t *testing.T) {
	x := []interface{}{nil}
	y := MustAnything(x).([]interface{})
	if !reflect.DeepEqual(x, y) || len(y) != 1 {
		t.Errorf("expect %v == %v; y had length %v (expected 1)", x, y, len(y))
	}
	var a interface{}
	b := MustAnything(a)
	if a != b {
		t.Errorf("expected %v == %v", a, b)
	}
}

func ExampleAvoidInfiniteLoops() {
	x := &Foo{
		Bar: 4,
	}
	x.Foo = x
	y := MustAnything(x).(*Foo)
	fmt.Printf("x == y: %v\n", x == y)
	fmt.Printf("x == x.Foo: %v\n", x == x.Foo)
	fmt.Printf("y == y.Foo: %v\n", y == y.Foo)
	// Output:
	// x == y: false
	// x == x.Foo: true
	// y == y.Foo: true
}

func TestUnsupportedKind(t *testing.T) {
	x := func() {}

	tests := []interface{}{
		x,
		map[bool]interface{}{true: x},
		[]interface{}{x},
	}

	for _, test := range tests {
		y, err := Anything(test)
		if y != nil {
			t.Errorf("expected %v to be nil", y)
		}
		if err == nil {
			t.Errorf("expected err to not be nil")
		}
	}
}

func TestUnsupportedKindPanicsOnMust(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected a panic; didn't get one")
		}
	}()
	x := func() {}
	MustAnything(x)
}

func TestMismatchedTypesFail(t *testing.T) {
	tests := []struct {
		input interface{}
		kind  reflect.Kind
	}{
		{
			map[int]int{1: 2, 2: 4, 3: 8},
			reflect.Map,
		},
		{
			[]int{2, 8},
			reflect.Slice,
		},
	}
	for _, test := range tests {
		for kind, copier := range copiers {
			if kind == test.kind {
				continue
			}
			actual, err := copier(test.input, nil, nil)
			if actual != nil {

				t.Errorf("%v attempted value %v as %v; should be nil value, got %v", test.kind, test.input, kind, actual)
			}
			if err == nil {
				t.Errorf("%v attempted value %v as %v; should have gotten an error", test.kind, test.input, kind)
			}
		}
	}
}

func TestTwoNils(t *testing.T) {
	type Foo struct {
		A int
	}
	type Bar struct {
		B int
	}
	type FooBar struct {
		Foo  *Foo
		Bar  *Bar
		Foo2 *Foo
		Bar2 *Bar
	}

	src := &FooBar{
		Foo2: &Foo{1},
		Bar2: &Bar{2},
	}

	dst := MustAnything(src)

	if !reflect.DeepEqual(src, dst) {
		t.Errorf("expect %v == %v; ", src, dst)
	}

}

type TestNilSliceAndMap struct {
	Slice []int
	Map   map[int]int
}

func TestStructWithNilSliceAndMAp(*testing.T) {
	x := &TestNilSliceAndMap{
		Slice: nil,
		Map:   nil,
	}
	y := MustAnything(x).(*TestNilSliceAndMap)
	fmt.Println(y)

	// Output:
	// <nil>
	// <nil>
}

func TestErrorFieldNotCopyable(t *testing.T) {
	x := time.Date(2021, time.November, 5, 21, 3, 35, 12, time.UTC)
	y, err := Anything(x)
	if err == nil || y != nil {
		t.Fatalf("Should have an error")
	}
	strError := `can't copy type time.Time cause of field wall`
	if err.Error() != strError {
		t.Fatalf("Wrong error got %#v want %#v", err.Error(), strError)
	}
}

package envfile

import (
	"bytes"
	"reflect"
	"testing"
)

var marshalCases = []struct {
	Name   string
	Input  interface{}
	Output []byte
	Error  error
}{
	{
		Name: "single string field",
		Input: struct {
			Test string
		}{
			Test: "abc123",
		},
		Output: []byte("TEST=abc123\n"),
	},
	{
		Name: "single tagged string field",
		Input: struct {
			Test string `env:"TEST_THING"`
		}{
			Test: "abc123",
		},
		Output: []byte("TEST_THING=abc123\n"),
	},
	{
		Name: "whitespace in value",
		Input: struct {
			Test string `env:"TEST"`
		}{
			Test: "abc123  ",
		},
		Output: []byte("TEST=abc123  \n"),
	},
	{
		Name: "tagged unsupported field in struct",
		Input: struct {
			Test int `env:"TEST"`
		}{
			Test: 1,
		},
		Output: []byte(""),
		Error:  ErrorUnsupportedType{reflect.Int},
	},
	{
		Name: "single tagged string field with omitempty",
		Input: struct {
			Test string `env:"TEST_THING,omitempty"`
		}{
			Test: "",
		},
		Output: []byte{},
	},
	{
		Name: "ignored unsupported field in struct",
		Input: struct {
			Bla  string `env:"BLA"`
			Test int    `env:"-"`
		}{
			Bla:  "Blabla",
			Test: 1,
		},
		Output: []byte("BLA=Blabla\n"),
	},
	{
		Name:   "marshal a nil value",
		Input:  nil,
		Output: []byte(""),
	},
	{
		Name:   "empty struct is passed as input",
		Input:  struct{}{},
		Output: []byte(""),
	},
	{
		Name:   "no struct is passed as input",
		Input:  "blablabla",
		Output: []byte(""),
		Error:  ErrorUnsupportedType{reflect.String},
	},
}

func TestMarshal(t *testing.T) {
	for _, c := range marshalCases {
		got, err := Marshal(c.Input)
		if err != c.Error {
			t.Errorf("[%s] error did not match, want: %v, got %v",
				c.Name, c.Error, err)
		}
		if bytes.Compare(c.Output, got) != 0 {
			t.Errorf("[%s] output did not match\nwant:\n%q,\tgot\n%q",
				c.Name, c.Output, got)
		}
	}
}

var unmarshalCases = []struct {
	Name   string
	Input  []byte
	Output interface{}
	Error  error
}{
	{
		Name:  "single string field",
		Input: []byte("TEST=abc123\n"),
		Output: struct {
			Test string
		}{
			Test: "abc123",
		},
	},
	{
		Name:  "single tagged string field",
		Input: []byte("TEST=abc123\n"),
		Output: struct {
			Test string `env:"TEST"`
		}{
			Test: "abc123",
		},
	},
	{
		Name: "multiple string field",
		Input: []byte(`TEST=abc123
FOO_VAR=foofoo
BAR_WHERE=bar123
`),
		Output: struct {
			Test string `env:"TEST"`
			Foo  string `env:"FOO_VAR"`
			Bar  string `env:"BAR_WHERE"`
		}{
			Test: "abc123",
			Foo:  "foofoo",
			Bar:  "bar123",
		},
	},
	{
		Name: "multiple string field with comments",
		Input: []byte(`#test
TEST=abc123
# foo variable
FOO_VAR=foofoo
BAR_WHERE=bar123
`),
		Output: struct {
			Test string `env:"TEST"`
			Foo  string `env:"FOO_VAR"`
			Bar  string `env:"BAR_WHERE"`
		}{
			Test: "abc123",
			Foo:  "foofoo",
			Bar:  "bar123",
		},
	},
	{
		Name: "multiple string field with empty lines",
		Input: []byte(`
TEST=abc123

FOO_VAR=foofoo
BAR_WHERE=bar123

`),
		Output: struct {
			Test string `env:"TEST"`
			Foo  string `env:"FOO_VAR"`
			Bar  string `env:"BAR_WHERE"`
		}{
			Test: "abc123",
			Foo:  "foofoo",
			Bar:  "bar123",
		},
	},
	{
		Name: "multiple string field with whitespace",
		Input: []byte(`TEST=abc123

FOO_VAR=foofoo
BAR_WHERE=bar123

`),
		Output: struct {
			Test string `env:"TEST"`
			Foo  string `env:"FOO_VAR"`
			Bar  string `env:"BAR_WHERE"`
		}{
			Test: "abc123",
			Foo:  "foofoo",
			Bar:  "bar123",
		},
	},
	{
		Name: "invalid variable line",
		Input: []byte(`TEST=abc123
FOO_VAR-foofoo
BAR_WHERE=bar123
`),
		Output: struct {
			Test string `env:"TEST"`
			Foo  string `env:"FOO_VAR"`
			Bar  string `env:"BAR_WHERE"`
		}{},
		Error: ErrorLineParsing{2},
	},
	{
		Name:  "lines with equal in value",
		Input: []byte("TEST=abc123=123=a\n"),
		Output: struct {
			Test string `env:"TEST"`
		}{
			Test: "abc123=123=a",
		},
	},
	{
		Name:  "target struct contains tagged int value",
		Input: []byte("TEST=1\n"),
		Output: struct {
			Test int `env:"TEST"`
		}{
			Test: 1,
		},
		Error: ErrorUnsupportedType{reflect.Int},
	},
	{
		Name:  "target struct contains ignored int value",
		Input: []byte("TEST=1\n"),
		Output: struct {
			Test int `env:"-"`
		}{},
		Error: nil,
	},
	{
		Name:  "target struct contains ignored string field",
		Input: []byte("TEST=abc\n"),
		Output: struct {
			Test string `env:"-"`
		}{},
		Error: nil,
	},
	{
		Name:  "target struct contains omitempty string field",
		Input: []byte("TEST=\n"),
		Output: struct {
			Test string `env:",omitempty"`
		}{},
		Error: nil,
	},
}

func TestUnmarshall(t *testing.T) {
	for _, c := range unmarshalCases {
		// Reflect a new pointer to the same type as the (anonymous)
		// struct in the output case.
		var got = reflect.New(reflect.TypeOf(c.Output))
		err := Unmarshal(c.Input, got.Interface())
		if err != c.Error {
			t.Errorf("[%s] error did not match, wanted error: %v, got %v", c.Name, c.Error, err)
		}
		if err != nil {
			// Do not test the output for error cases.
			continue
		}
		// Get the struct value that our got pointer is pointing to.
		gotValue := reflect.Indirect(got).Interface()
		if !reflect.DeepEqual(c.Output, gotValue) {
			t.Errorf("[%s] output does not match\nwant:\n%+v,\tgot\n%+v", c.Name, c.Output, gotValue)
		}
	}
}

func TestUnmarshalIntoNil(t *testing.T) {
	err := Unmarshal([]byte("TEST=123"), nil)
	if err == nil {
		t.Errorf("unmarshal into nil did not return an error")
	}
}

func TestErrors(t *testing.T) {
	var (
		err  error
		want string
	)
	err = ErrorUnsupportedType{reflect.Int}
	want = "unsupported type int"
	if err.Error() != want {
		t.Errorf("error did not match, want %q, got %q", want, err.Error())
	}
	err = ErrorLineParsing{5}
	want = "error parsing line 5"
	if err.Error() != want {
		t.Errorf("error did not match, want %q, got %q", want, err.Error())
	}
}

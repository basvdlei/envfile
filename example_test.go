package envfile

import "fmt"

func ExampleMarshal() {
	values := struct {
		Name      string
		MySetting string `env:"MY_SETTING"`
		Empty     string `env:",omitempty"`
	}{
		Name:      "foo",
		MySetting: "https://127.0.0.1",
	}
	out, err := Marshal(values)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(string(out))
	// Output: NAME=foo
	// MY_SETTING=https://127.0.0.1
}

func ExampleUnmarshal() {
	data := []byte(`FOO=bar
DB=test
# Comments and empty lines are ignored
EMPTY=

IGNORED=valuenotread
`)

	v := struct {
		Foo      string
		Database string `env:"DB"`
		Empty    string `env:",omitempty"`
		Ignored  string `env:"-"`
	}{}
	err := Unmarshal(data, &v)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println("Foo=" + v.Foo)
	fmt.Println("Database=" + v.Database)
	fmt.Println("Empty=" + v.Empty)
	fmt.Println("Ignored=" + v.Ignored)
	// Output: Foo=bar
	// Database=test
	// Empty=
	// Ignored=
}

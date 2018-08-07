// Package envfile contains functions to (un)marshal EnvironmentFile to/from Go
// types.
//
// This package is using the same 'style' as the the stardard library
// encoding/json package for (un)marshaling EnvironmentFile (dot env) syntax.
package envfile

import (
	"bufio"
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

// ErrorUnsupportedType is returned when the value is or contains unsupported
// types.
type ErrorUnsupportedType struct {
	Kind reflect.Kind
}

// Error implements the error interface.
func (e ErrorUnsupportedType) Error() string {
	return fmt.Sprintf("unsupported type %v", e.Kind)
}

// ErrorLineParsing is returned when a env line can not be parsed.
type ErrorLineParsing struct {
	LineNumber int
}

// Error implements the error interface.
func (e ErrorLineParsing) Error() string {
	return fmt.Sprintf("error parsing line %d", e.LineNumber)
}

// Marshal returns the EnvironmentFile encoding of v.
//
// The "omitempty" option specifies that the field should be omitted from the
// encoding if the field has an empty value.
//
// Examples of struct field tags:
//
//   // Field appears in EnvironmentFile as key "MY_NAME".
//   Field string `env:"MY_NAME"`
//
//   // Field appears in EnvironmentFile as key "FIELD".
//   Field string`
//
//   // Field appears in JSON as key "myName" and
//   // the field is omitted from the object if its value is empty,
//   // as defined above.
//   Field string `json:"myName,omitempty"`
//
//   // Field appears in JSON as key "Field" (the default), but
//   // the field is skipped if empty.
//   // Note the leading comma.
//   Field int `json:",omitempty"`
//
// Only string fields are supported and it will return a ErrorUnsupportedType
// when fields with other types are not explicilty ignored.
func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	t := reflect.TypeOf(v)
	if t == nil {
		return []byte{}, nil
	}
	if k := t.Kind(); k != reflect.Struct {
		return []byte{}, ErrorUnsupportedType{k}
	}
	val := reflect.ValueOf(v)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		keyname, opts := parseFieldOpts(field)
		if opts.Skip {
			continue
		}
		switch t.Field(i).Type.Kind() {
		case reflect.String:
			if !(opts.OmitEmpty && val.Field(i).String() == "") {
				fmt.Fprintf(&buf, "%s=%s\n", keyname, val.Field(i))
			}
		default:
			return []byte{}, ErrorUnsupportedType{t.Field(i).Type.Kind()}
		}
	}
	return buf.Bytes(), nil
}

// Unmarshal parses the environmentfile encoded data and stores the result in
// the value pointed to by v.
func Unmarshal(data []byte, v interface{}) error {
	r := bytes.NewReader(data)
	scanner := bufio.NewScanner(r)
	count := 0
	for scanner.Scan() {
		count++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			return ErrorLineParsing{count}
		}
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			return ErrorUnsupportedType{rv.Kind()}
		}
		t := reflect.TypeOf(v).Elem()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			keyname, opts := parseFieldOpts(field)
			if opts.Skip {
				continue
			}
			if strings.TrimSpace(kv[0]) == keyname {
				field := rv.Elem().Field(i)
				switch field.Kind() {
				case reflect.String:
					if !(opts.OmitEmpty && kv[1] == "") {
						field.SetString(strings.TrimSpace(kv[1]))
					}
				default:
					return ErrorUnsupportedType{field.Kind()}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// envOptions contains the options set in the field.
type envOptions struct {
	Skip      bool
	OmitEmpty bool
}

// parseFieldOpts will convert a StructType field tag to an environment name.
func parseFieldOpts(field reflect.StructField) (name string, opts envOptions) {
	tag := field.Tag.Get("env")
	options := strings.Split(tag, ",")
	if len(options) > 1 {
		for _, v := range options[1:] {
			switch v {
			case "omitempty":
				opts.OmitEmpty = true
			}
		}
	}
	switch options[0] {
	case "-":
		opts.Skip = true
	case "":
		// TODO filter out 'bad' characters
		name = strings.ToUpper(field.Name)
	default:
		// TODO: Make sure the specified variable name is valid.
		name = options[0]
	}
	return
}

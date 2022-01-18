package errors_test

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/eluv-io/errors-go"
)

func ExampleError() {
	// Single error.
	e1 := errors.E("get", errors.K.IO, io.EOF)
	fmt.Println("\nSimple error:")
	fmt.Println(e1)

	// Nested error.
	fmt.Println("\nNested error:")
	e2 := errors.E("read", e1)
	fmt.Println(e2)

	// Output:
	//
	// Simple error:
	// op [get] kind [I/O error] cause [EOF]
	//
	// Nested error:
	// op [read] kind [I/O error] cause:
	// 	op [get] kind [I/O error] cause [EOF]
}

func ExampleTemplate() {
	var err error
	e := errors.Template("validate", errors.K.Invalid)

	// add fields
	err = e("reason", "invalid character", "character", "$")
	fmt.Println(err)

	// override kind and set cause
	err = e().WithKind(errors.K.IO).WithCause(io.EOF)
	fmt.Println(err)

	// Overriding kind and setting cause also works by simply passing them as
	// args to the template function...
	err = e(errors.K.IO, io.EOF)
	fmt.Println(err)

	// IfNotNil() returns an error only if the provided cause is not nil
	err = e.IfNotNil(nil)
	fmt.Println(err)

	e = e.Add("sub", "validateNestedData")
	err = e("reason", "missing hash")
	fmt.Println(err)

	// Output:
	//
	// op [validate] kind [invalid] reason [invalid character] character [$]
	// op [validate] kind [I/O error] cause [EOF]
	// op [validate] kind [I/O error] cause [EOF]
	// <nil>
	// op [validate] kind [invalid] sub [validateNestedData] reason [missing hash]
}

func ExampleKind_Default() {
	e := errors.Template("validate", errors.K.Invalid.Default())

	nested1 := errors.E(io.EOF)
	fmt.Println(e(nested1)) // kind [invalid]

	nested2 := errors.E(errors.K.IO, io.EOF)
	fmt.Println(e(nested2)) // kind [I/O error]

	// Output:
	//
	// op [validate] kind [invalid] cause:
	// 	kind [unclassified error] cause [EOF]
	// op [validate] kind [I/O error] cause:
	// 	kind [I/O error] cause [EOF]
}

func ExampleMatch() {
	err := errors.Str("network unreachable")

	// Construct an error, one we pretend to have received from a test.
	got := errors.E("get", errors.K.IO, err)

	// Now construct a reference error, which might not have all
	// the fields of the error from the test.
	expect := errors.E().WithKind(errors.K.IO).WithCause(err)

	fmt.Println("Match:", errors.Match(expect, got))

	// Now one that's incorrect - wrong Kind.
	got = errors.E().WithOp("get").WithKind(errors.K.Permission).WithCause(err)

	fmt.Println("Mismatch:", errors.Match(expect, got))

	// Output:
	//
	// Match: true
	// Mismatch: false
}

func ExampleError_FormatError() {
	jsn := `{`
	e := errors.Template("parse", errors.K.Invalid, "account", 5, "json", jsn)

	var err error
	var user map[string]interface{}
	if err = json.Unmarshal([]byte(jsn), &user); err != nil {
		err = e(err, "reason", "failed to decode user")
	}

	fmt.Println(err)
	fmt.Println(errors.Wrap(err).FormatError(false, "op", "kind", "reason", "", "cause"))

	// Output:
	//
	// op [parse] kind [invalid] account [5] json [{] reason [failed to decode user] cause [unexpected end of JSON input]
	// op [parse] kind [invalid] reason [failed to decode user] account [5] json [{] cause [unexpected end of JSON input]
}

func ExampleError_Error() {
	defer resetDefaultFieldOrder()()

	err := errors.E("get user", errors.K.IO, io.EOF, "account", "acme", "user", "joe")
	fmt.Println(err)

	errors.DefaultFieldOrder = []string{"op", "kind", "", "cause"} // same as default (nil)
	fmt.Println(err)

	errors.DefaultFieldOrder = []string{"kind"}
	fmt.Println(err)

	errors.DefaultFieldOrder = []string{"op", "user"}
	fmt.Println(err)

	errors.DefaultFieldOrder = []string{"op", "cause"}
	fmt.Println(err)

	fmt.Println()
	fmt.Println("Nested:")
	err = errors.E("get info", errors.K.Invalid, err)

	errors.DefaultFieldOrder = nil
	fmt.Println(err)

	// not putting the "cause" last is a bad idea with nested errors...
	errors.DefaultFieldOrder = []string{"op", "cause"}
	fmt.Println(err)

	// Output:
	//
	// op [get user] kind [I/O error] account [acme] user [joe] cause [EOF]
	// op [get user] kind [I/O error] account [acme] user [joe] cause [EOF]
	// kind [I/O error] op [get user] account [acme] user [joe] cause [EOF]
	// op [get user] user [joe] kind [I/O error] account [acme] cause [EOF]
	// op [get user] cause [EOF] kind [I/O error] account [acme] user [joe]
	//
	// Nested:
	// op [get info] kind [invalid] cause:
	// 	op [get user] kind [I/O error] account [acme] user [joe] cause [EOF]
	// op [get info] cause:
	// 	op [get user] cause [EOF] kind [I/O error] account [acme] user [joe] kind [invalid]

}

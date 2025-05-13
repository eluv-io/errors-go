package errors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	stderrors "errors"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
)

// populateStacktrace controls whether stacktraces are captured on error creation per default or not. This is
// (obviously) a runtime setting - use the "errnostack" build tag to disable stacktrace captures at compile time.
var populateStacktrace = atomic.Bool{}

func init() {
	SetPopulateStacktrace(true)
}

func SetPopulateStacktrace(b bool) {
	populateStacktrace.Store(b)
}

func PopulateStacktrace() bool {
	return populateStacktrace.Load()
}

// PrintStacktrace controls whether stacktraces are printed per default or not.
var PrintStacktrace = true

// PrintStacktracePretty enables additional formatting of stacktraces by aligning functions to the longest source filename.
//
// Pretty print:
//
//	github.com/eluv-io/errors-go/stack_with_long_filename_test.go:6 createErrorWithExtraLongFilename()
//	github.com/eluv-io/errors-go/stack_test.go:108                  func4()
//	github.com/eluv-io/errors-go/stack_test.go:109                  func4()
//	github.com/eluv-io/errors-go/stack_test.go:104                  T.func3()
//	github.com/eluv-io/errors-go/stack_test.go:99                   T.func2()
//	github.com/eluv-io/errors-go/stack_test.go:90                   func1()
//	github.com/eluv-io/errors-go/stack_test.go:47                   TestStack()
//	testing/testing.go:909                                          tRunner()
//	runtime/asm_amd64.s:1357                                        goexit()
//
// Regular:
//
//	github.com/eluv-io/errors-go/stack_with_long_filename_test.go:6	createErrorWithExtraLongFilename()
//	github.com/eluv-io/errors-go/stack_test.go:108	func4()
//	github.com/eluv-io/errors-go/stack_test.go:109	func4()
//	github.com/eluv-io/errors-go/stack_test.go:104	T.func3()
//	github.com/eluv-io/errors-go/stack_test.go:99	T.func2()
//	github.com/eluv-io/errors-go/stack_test.go:90	func1()
//	github.com/eluv-io/errors-go/stack_test.go:47	TestStack()
//	testing/testing.go:909	tRunner()
//	runtime/asm_amd64.s:1357	goexit()
var PrintStacktracePretty = true

// MarshalStacktrace controls whether stacktraces are marshaled to JSON or not. If enabled, an extra "stacktrace" field
// is added to the error's JSON struct.
var MarshalStacktrace = true

// MarshalStacktraceAsArray controls whether stacktraces are marshaled to JSON as a single string blob or as JSON array
// containing the individual lines of the stacktrace.
var MarshalStacktraceAsArray = true

// DefaultFieldOrder defines the default order of an Error's fields in its String and JSON representations.
//
// The first empty string "" acts as alias for all fields that are unreferenced in the field order slice. They are
// printed in the order they were added in E() and With().
//
// Trailing field keys after "" will be printed after any unreferenced fields.
//
// "op", "kind" and "cause" fields are specially treated if they don't appear in the order slice. "op" and "kind" will
// be printed first among the unreferenced fields. "cause" will be printed last (even after all trailing fields).
// Hence the default nil is equivalent to []string{"op", "kind", "", "cause"}
var DefaultFieldOrder []string = nil

// Error is the type that implements the error interface and which is returned by E(), NoTrace(), etc.
type Error struct {
	// the operation
	op string
	// the error kind (or type)
	kind Kind
	// the default kind - only used if kind is not set
	defaultKind Kind
	// the optional cause
	cause error
	// additional fields
	fields orderedMap
	// Stack information; not used if the 'errnostack' build tag is set.
	stack
	// ignoreStack is set to true when an error is unmarshalled from JSON (because the stack field does not correspond
	// to the stack location where the error actually occurred, but rather where it was unmarshalled)
	ignoreStack bool
	// the stacktrace from an unmarshalled error (if any)
	unmarshalledStacktrace string
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// MarshalJSON marshals this error as a JSON object
func (e *Error) MarshalJSON() ([]byte, error) {
	return e.marshalFields(true)
}

func (e *Error) marshalFields(marshalStack bool) (res []byte, err error) {
	b := &bytes.Buffer{}
	needSep := false

	kv := func(key interface{}, val interface{}) error {
		if needSep {
			b.WriteByte(',')
		}
		needSep = true

		bts, err := json.Marshal(key)
		if err != nil {
			return err
		}
		b.Write(bts)
		b.WriteByte(':')

		if key == "cause" {
			switch cause := val.(type) {
			case *Error:
				bts, err = cause.marshalFields(false)
			default:
				val, _ = convertForJSONMarshalling(cause)
				bts, err = json.Marshal(val)
			}
		} else {
			bts, err = json.Marshal(val)
		}

		if err != nil {
			return err
		}
		b.Write(bts)
		return nil
	}

	b.WriteByte('{')

	err = e.writeFields(DefaultFieldOrder, kv)
	if err != nil {
		return nil, err
	}

	if marshalStack && MarshalStacktrace && !e.ignoreStack && e.hasStack() {
		bb := new(bytes.Buffer)
		e.printStack(bb)
		if MarshalStacktraceAsArray {
			err = kv("stacktrace", stacktraceToArray(bb.String()))
		} else {
			err = kv("stacktrace", bb.String())
		}
		if err != nil {
			return nil, err
		}
	}

	b.WriteByte('}')

	return b.Bytes(), nil
}

func stacktraceToArray(s string) []string {
	// trim empty lines or lines containing only whitespace
	s = strings.Trim(s, "\n\t ")
	if s == "" {
		return []string{}
	}

	res := strings.Split(s, "\n")
	for i, line := range res {
		res[i] = strings.Trim(line, "\t\n ")
	}
	return res
}

// UnmarshalJSON unmarshals the given JSON text, retaining the order of fields according to the JSON structure.
func (e *Error) UnmarshalJSON(b []byte) error {
	fields := make(map[orderedKey]valOrMap)
	err := json.Unmarshal(b, &fields)
	if err != nil {
		return err
	}
	e.unmarshalFrom(fields)
	return nil
}

func (e *Error) unmarshalFrom(f map[orderedKey]valOrMap) {
	keys := make(ordereKeys, 0, len(f))
	for key := range f {
		keys = append(keys, key)
	}
	sort.Sort(keys)
	e.fields.Grow(len(f))
	for _, key := range keys {
		val := f[key].Get()
		if key.key == "stacktrace" {
			slice, ok := val.([]interface{})
			if ok {
				sb := strings.Builder{}
				for _, line := range slice {
					if sb.Len() > 0 {
						sb.WriteString("\n")
					}
					sb.WriteString("\t")
					sb.WriteString(toString(line))
				}
				e.unmarshalledStacktrace = sb.String()
			} else {
				e.unmarshalledStacktrace = toString(val)
			}
		} else {
			_ = e.With(key.key, val)
		}
	}
}

// Op returns the error's operation or "" if no op is set.
func (e *Error) Op() string {
	return e.op
}

// Kind returns the error's kind or errors.K.Other if no kind is set.
func (e *Error) Kind() Kind {
	return e.effectiveKind(K.Other)
}

// Cause returns the error's cause or nil if no cause is set.
func (e *Error) Cause() error {
	return e.cause
}

// WithOp sets the given operation and returns this error instance for call chaining.
func (e *Error) WithOp(op string) *Error {
	if op != "" {
		e.op = op
	}
	return e
}

// WithKind sets the given kind and returns this error instance for call chaining.
func (e *Error) WithKind(kind Kind) *Error {
	if kind != "" {
		e.kind = kind
	}
	return e
}

// WithDefaultKind sets the given kind as default and returns this error instance for call chaining. The default kind is
// only used if the kind is not otherwise set e.g. with an explicit call to Error.Kind(kind) or by inheriting it from a
// nested error. It's equivalent to calling Error.With(kind.Default()).
func (e *Error) WithDefaultKind(kind Kind) *Error {
	e.defaultKind = kind
	return e
}

// WithCause sets the given original error and returns this error instance for call chaining. If the cause is an *Error
// and this error's kind is not yet initialized, it inherits the kind of the cause.
func (e *Error) WithCause(err error) *Error {
	if err != nil {
		e.cause = err
	}
	return e
}

// With adds additional context information in the form of key value pairs and returns this error instance for call
// chaining.
func (e *Error) With(args ...interface{}) *Error {
	argc := len(args)

	if argc == 1 {
		if slice, ok := args[0].([]interface{}); ok {
			// there is a single argument, and it's an []interface{}... most probably the caller forgot to specify the
			// ellipsis in the call invocation: With(msg, slice...). Hence we treat the slice as the fields.
			args = slice
			argc = len(slice)
		}
	}
	if argc == 0 {
		return e
	}

	e.fields.Grow(argc)
	for idx := 0; idx < argc; idx++ {
		key := args[idx]

		if key == nil {
			continue
		}

		switch a := key.(type) {
		case Kind:
			_ = e.WithKind(a)
			continue
		case DefaultKind:
			_ = e.WithDefaultKind(Kind(a))
			continue
		case error:
			_ = e.WithCause(a)
			continue
		}

		var val interface{}
		hasVal := idx+1 < argc
		if hasVal {
			val = args[idx+1]
			idx++

			switch key {
			case "op":
				if op, ok := val.(string); ok {
					_ = e.WithOp(op)
				}
				continue
			case "kind":
				knd, ok := val.(Kind)
				if !ok {
					knd = Kind(toString(val))
				}
				_ = e.WithKind(knd)
				continue
			case "cause":
				if val == nil {
					continue
				}
				cause, ok := val.(error)
				if !ok {
					cause = Str(toString(val))
				}
				_ = e.WithCause(cause)
				continue
			}
		}

		if hasVal {
			e.fields.Append(key, val)
		} else {
			e.fields.Append(key)
		}
	}
	return e
}

func (e *Error) isZero() bool {
	return e.op == "" && e.kind == "" && e.cause == nil && len(e.fields) == 0
}

func (e *Error) field(key string) (interface{}, bool) {
	switch key {
	case "op":
		if e.op != "" {
			return e.op, true
		}
		return nil, false
	case "kind":
		return e.Kind(), true
	case "cause":
		if e.cause != nil {
			return e.cause, true
		}
		return nil, false
	}
	return e.fields.Get(key)
}

// Field returns the given field from this error or any nested errors. Returns nil if the field does not exist.
func (e *Error) Field(key string) interface{} {
	var err interface{} = e
	for {
		ex, ok := err.(*Error)
		if !ok {
			break
		}
		val, ok := ex.field(key)
		if ok {
			return val
		}
		err = ex.cause
	}
	return nil
}

// GetField attempts to retrieve the field with the given key in this Error and returns its value converted to a string
// with fmt.Sprint(val). If the field doesn't exist, it tries to find it (recursively) in the 'cause' of the error.
// Returns the retrieved field value and true if found, or the empty string and false if not found.
func (e *Error) GetField(key string) (string, bool) {
	var err interface{} = e
	for {
		ex, ok := err.(*Error)
		if !ok {
			break
		}
		val, ok := ex.field(key)
		if ok {
			return toString(val), true
		}
		err = ex.cause
	}
	return "", false
}

// GetField returns the result of calling the GetField() method on the given err if it is an *Error. Returns "", false
// otherwise.
func GetField(err error, key string) (string, bool) {
	e, ok := err.(*Error)
	if !ok {
		return "", false
	}
	return e.GetField(key)
}

// Field returns the result of calling the Field() method on the given err if it is an *Error. Returns nil otherwise.
func Field(err error, key string) interface{} {
	e, ok := err.(*Error)
	if !ok {
		return nil
	}
	return e.Field(key)
}

// Separator is the string used to separate nested errors. By default, nested errors
// are indented on a new line.
var Separator = ":\n\t"

// E creates a new error initialized with the given (optional) operation, kind, cause and key-value fields. All
// arguments are optional, but if provided they have to be specified in that order.
//
// If no kind is specified, the kind of the cause is used if available. Otherwise, K.Other is assigned.
//
// The op should specify the operation that failed, e.g. "download" or "load config". It should not be an error
// "message" like "download failed" or "failed to load config" - the fact that the operation has failed is implied by
// the error itself.
//
// Examples:
//
//	errors.E()                             --> error with kind set to errors.K.Other, all other fields empty
//	errors.E("download")                   --> same as errors.E().WithOp("download")
//	errors.E("download", errors.K.IO)      --> same as errors.E().WithOp("download").WithKind(errors.K.IO)
//	errors.E("download", errors.K.IO, err) --> same as errors.E().WithOp("download").WithKind(errors.K.IO).WithCause(err)
//	errors.E("download", errors.K.IO, err, "file", f, "user", usr) --> same as errors.E()...With("file", f).With("user", usr)
//	errors.E(errors.K.NotExist, "file", f) --> same as errors.E().WithKind(errors.K.NotExist).With("file", f)
func E(args ...interface{}) *Error {
	e := NoTrace(args...)

	if PopulateStacktrace() {
		e.populateStack()
	}

	return e
}

// NoTrace is the same as E, but does not populate a stack trace. Use in cases where the stacktrace is not desired.
func NoTrace(args ...interface{}) *Error {
	e := &Error{}
	argc := len(args)

	if argc == 1 {
		if slice, ok := args[0].([]interface{}); ok {
			// there is a single argument, and it's an []interface{}... most probably the caller forgot to specify the
			// ellipsis in the call invocation: With(msg, slice...). Hence we treat the slice as the fields.
			args = slice
			argc = len(slice)
		}
	}

	if len(args) > 0 {
		if op, ok := args[0].(string); ok {
			// the first arg is a string - use it as op
			_ = e.WithOp(op)
			args = args[1:]
		}
	}

	_ = e.With(args...)

	return e
}

// Template returns a function that creates a base error with an initial set of fields. When called, additional fields
// can be passed that complement the error template:
//
//	e := errors.Template("unmarshal", K.Invalid)
//	...
//	if invalid {
//	  return e("reason", "invalid format")
//	}
//	...
//	if err != nil {
//	  return e(err)
//	}
func Template(fields ...interface{}) TemplateFn {
	return func(f ...interface{}) *Error {
		return E(append(fields, f...)...).dropStackFrames(1)
	}
}

// T is an alias for Template.
func T(fields ...interface{}) TemplateFn {
	return Template(fields...)
}

// TemplateNoTrace is like Template but produces an error without stacktrace information.
func TemplateNoTrace(fields ...interface{}) TemplateFn {
	return func(f ...interface{}) *Error {
		return NoTrace(append(fields, f...)...)
	}
}

// TNoTrace is an alias for TemplateNoTrace
func TNoTrace(fields ...interface{}) TemplateFn {
	return TemplateNoTrace(fields...)
}

type TemplateFn func(fields ...interface{}) *Error

// IfNotNil returns an error based on this template iff 'err' is not nil. Otherwise returns nil.
func (t TemplateFn) IfNotNil(err error, fields ...interface{}) error {
	if err == nil {
		return nil
	}
	return t(append(fields, err)...).dropStackFrames(1)
}

// Add adds additional fields to this template.
func (t TemplateFn) Add(fields ...interface{}) TemplateFn {
	return func(f ...interface{}) *Error {
		return t(append(fields, f...)...).dropStackFrames(1)
	}
}

// Fields returns the fields of the template, excluding "op", "kind" and "cause", appended with the given fields. This
// is useful in situations where the same information is used for logging and in errors.
func (t TemplateFn) Fields(moreFields ...interface{}) []interface{} {
	return append(t().fields, moreFields...)
}

// String returns a string representation of this template.
func (t TemplateFn) String() string {
	return t().Error()
}

// MarshalJSON marshals this template to JSON.
func (t TemplateFn) MarshalJSON() ([]byte, error) {
	return t().MarshalJSON()
}

type TFn func(err error, fields ...interface{}) error

// Error returns the string presentation of this Error. Fields are ordered according to DefaultFieldOrder. The
// stacktrace is printed if available.
func (e *Error) Error() string {
	return e.toString(true)
}

// ErrorNoTrace returns the error as string just like Error() but omits the stack trace.
func (e *Error) ErrorNoTrace() string {
	return e.toString(false)
}

func (e *Error) toString(printStacktrace bool, fieldOrder ...string) string {
	if e == nil {
		return ""
	}

	b := new(bytes.Buffer)

	if len(fieldOrder) == 0 {
		fieldOrder = DefaultFieldOrder
	}
	_ = e.writeFields(fieldOrder, func(key interface{}, val interface{}) error {
		e.writeKeyVal(b, key, val)
		return nil
	})

	if printStacktrace && PrintStacktrace && !e.ignoreStack && e.hasStack() {
		_, _ = fmt.Fprint(b, "\n")
		e.printStack(b)
	}
	return b.String()
}

func (e *Error) writeFields(fieldOrder []string, writeKV func(key interface{}, val interface{}) error) (err error) {
	// returns true for fields that are not listed in fieldOrder
	unreferenced := func(key interface{}) bool {
		if key == "stacktrace" && !PrintStacktrace {
			return false
		}
		// iterating over the fieldOrder slice is faster than using a map with up to 10 entries in fieldOrder, which
		// practically never occurs.
		for _, field := range fieldOrder {
			if field == key {
				return false
			}
		}
		return true
	}

	printOthers := true
	printOtherFields := func() (err error) {
		if !printOthers {
			return
		}
		printOthers = false
		if e.op != "" && unreferenced("op") {
			err = writeKV("op", e.op)
		}
		if unreferenced("kind") {
			err = writeKV("kind", e.Kind())
		}
		if err != nil {
			return err
		}
		for i := 0; i+1 < len(e.fields); i += 2 {
			key := e.fields[i]
			if unreferenced(key) {
				err = writeKV(key, e.fields[i+1])
				if err != nil {
					return err
				}
			}
		}
		if e.cause != nil && unreferenced("cause") {
			err = writeKV("cause", e.cause)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, key := range fieldOrder {
		if key == "" {
			err = printOtherFields()
		} else if val, ok := e.field(key); ok {
			err = writeKV(key, val)
		}
		if err != nil {
			return err
		}
	}

	return printOtherFields()
}

func (e *Error) writeKeyVal(b *bytes.Buffer, key interface{}, val interface{}) {
	if key == "cause" {
		if cause, ok := e.cause.(*Error); ok {
			if !cause.isZero() {
				pad(b, " ")
				b.WriteString("cause")
				b.WriteString(Separator)
				b.WriteString(cause.toString(false))
			}
			return
		}
	}
	pad(b, " ")
	b.WriteString(key.(string))
	b.WriteString(" [")
	b.WriteString(fmt.Sprint(val))
	b.WriteString("]")
}

// ClearStacktrace creates a copy of this error and removes the stacktrace from it and all nested causes.
func (e *Error) ClearStacktrace() *Error {
	clone := *e
	clone.fields = make([]interface{}, len(e.fields))
	copy(clone.fields, e.fields)

	clone.clearStack()
	clone.fields.Delete("stacktrace")
	clone.fields.Delete("remote_stack")
	clone.unmarshalledStacktrace = ""

	if e2, ok := clone.cause.(*Error); ok {
		clone.cause = e2.ClearStacktrace()
	}
	return &clone
}

func (e *Error) effectiveKind(def Kind) Kind {
	if e.kind != "" {
		return e.kind
	}

	if e.defaultKind != "" {
		def = e.defaultKind
	} else if def == "" {
		def = K.Other
	}

	if e.cause != nil {
		var cause *Error
		if errors.As(e.cause, &cause) {
			eff := cause.effectiveKind(def)
			if eff != "" {
				return eff
			}
		}
	}

	return def
}

// FormatError converts this error to a string like String(), but prints fields according to the given field order. See
// DefaultFieldOrder for more information.
//
// A stacktrace (if available) is printed if printStack is true.
func (e *Error) FormatError(printStack bool, fieldOrder ...string) string {
	return e.toString(printStack, fieldOrder...)
}

// Str is an alias for the standard errors.New() function
func Str(text string) error {
	return stderrors.New(text)
}

// Match compares two errors.
//
// If one of the arguments is not of type *Error, Match will return reflect.DeepEqual(err1, err2).
//
// Otherwise it returns true iff every non-zero element of the first error is equal to the corresponding element of the
// second. If the Cause field is a *Error, Match recurs on that field; otherwise it compares the strings returned by the
// Error methods. Elements that are in the second argument but not present in the first are ignored.
//
// For example:
//
//	Match(errors.E("authorize", errors.Permission), err)
//
// tests whether err is an Error with op=authorize and kind=Permission.
func Match(err1, err2 error) bool {
	if err1 == nil {
		return err2 == nil
	}
	if err2 == nil {
		return false
	}

	e1, ok1 := err1.(*Error)
	e2, ok2 := err2.(*Error)
	if !ok1 {
		// comparing to a regular error
		if !ok2 {
			return reflect.DeepEqual(err1, err2)
		}
		return Match(err1, e2.cause)
	}
	if !ok2 {
		return false
	}

	if e1.op != "" && e1.op != e2.op {
		return false
	}
	if e1.kind != "" && e1.kind != e2.Kind() {
		return false
	}

	for i := 0; i+1 < len(e1.fields); i += 2 {
		key := e1.fields[i].(string)
		val1 := e1.fields[i+1]

		val2, ok := e2.fields.Get(key)
		if !ok {
			return false
		}

		var cause1, cause2 error
		if cause1, ok = val1.(error); ok {
			if cause2, ok = val2.(error); ok {
				return Match(cause1, cause2)
			}
			return false
		}
		if !reflect.DeepEqual(val1, val2) {
			return false
		}
	}

	if e1.cause != nil {
		return Match(e1.cause, e2.cause)
	}
	return true
}

// IsNotExist reports whether err is an *Error of Kind NotExist. Returns false if err is nil.
func IsNotExist(err error) bool {
	return IsKind(K.NotExist, err)
}

// IsKind reports whether err is an *Error of the given Kind.
// Returns false if err is nil.
func IsKind(expected Kind, err interface{}) bool {
	for {
		e, ok := err.(*Error)
		if !ok || e == nil {
			return false
		}
		if e.Kind() == expected {
			return true
		}
		err = e.cause
	}
}

// GetRoot returns the innermost nested *Error of the given error, or nil if the provided object is not an *Error.
func GetRoot(err interface{}) *Error {
	var root *Error
	for {
		e, ok := err.(*Error)
		if !ok {
			return root
		}
		root = e
		err = e.cause
	}
}

// GetRootCause returns the first nested error that is not an *Error object. Returns NilError if err is nil.
func GetRootCause(err error) error {
	for err != nil {
		e, ok := err.(*Error)
		if !ok {
			return err
		}
		if e.cause == nil {
			return NilError
		}
		err = e.cause
	}
	return NilError
}

// UnmarshalJsonErrorList unmarshals a list of errors. JSON objects are unmarshalled into Error objects, strings into
// generic errors created with Str(s). Empty objects or strings are ignored.
//
// The exact JSON structure is:
//
//	{
//	  "errors": [
//		{"op": "op1"},
//		"EOF",
//		{},
//		{"op": "op2"}
//	  ]
//	}
//
// Empty errors are removed from the returned list.
func UnmarshalJsonErrorList(bts []byte) (ErrorList, error) {
	list := ErrorList{}
	err := list.UnmarshalJSON(bts)
	return list, err
}

// ClearStacktrace removes the stacktrace from the given error if it's an instance of *Error. Does nothing otherwise.
func ClearStacktrace(err error) error {
	e, ok := err.(*Error)
	if !ok {
		return err
	}
	return e.ClearStacktrace()
}

// Ignore simply ignores a potential error returned by the given function.
//
// Useful in defer statements where the deferred function returns an error, i.e.
//
//	defer writer.Close()
//
// can be written as
//
//	defer errors.Ignore(writer.Close)
func Ignore(f func() error) {
	if f == nil {
		return
	}
	_ = f()
}

// Log calls the given function and logs the error if any. Prints errors to stdout if logFn is nil.
//
// Useful in defer statements where the deferred function returns an error, i.e.
//
//	defer writer.Close()
//
// can be written as
//
//	defer errors.Log(writer.Close, logFn)
func Log(f func() error, logFn func(msg string, fields ...interface{})) {
	if f == nil {
		return
	}

	err := f()
	if err == nil {
		return
	}

	msg := "errors.Log function call returned error"
	fnName := "unknown"
	if ffp := runtime.FuncForPC(reflect.ValueOf(f).Pointer()); ffp != nil {
		fnName = ffp.Name()
	}

	if logFn == nil {
		fmt.Printf("%s: function=%s error=%s\n", msg, fnName, err)
	} else {
		logFn(msg, "function", fnName, "error", err)
	}
}

// Wrap wraps the given error in an Error instance with E(err) if err is not an *Error itself - otherwise returns err as
// *Error unchanged. Returns nil if err is nil.
func Wrap(err error, args ...interface{}) *Error {
	if err == nil {
		return nil
	}
	e, ok := err.(*Error)
	if !ok {
		e = E(err)
	}
	if len(args) > 0 {
		_ = e.With(args...)
	}
	return e
}

// FromContext creates an error from the given context and additional error arguments as passed to E(). It returns
//   - nil if ctx.Err() returns nil
//   - an error from the given args and kind Timeout if the ctx timed out
//   - an error from the given args and kind Cancelled if the ctx was cancelled
//   - an error from the given args and the cause set to ctx.Err() otherwise.
func FromContext(ctx context.Context, args ...interface{}) *Error {
	if ctx == nil {
		return nil
	}
	switch ctx.Err() {
	case nil:
		return nil
	case context.DeadlineExceeded:
		return E(args...).WithKind(K.Timeout)
	case context.Canceled:
		return E(args...).WithKind(K.Cancelled)
	}
	return E(args...).WithCause(ctx.Err())
}

// TypeOf returns the type of the given value as string.
func TypeOf(val interface{}) string {
	return fmt.Sprintf("%T", val)
}

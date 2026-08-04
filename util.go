package errors

import (
	"bytes"
	"encoding"
	"encoding/json"
	"reflect"
)

// pad appends str to the buffer if the buffer already has some data.
func pad(b *bytes.Buffer, str string) {
	if b.Len() == 0 {
		return
	}
	b.WriteString(str)
}

// convertForJSONMarshalling replaces the given obj if it's a builtin "error" interface with its string representation
// (obj.Error()), because "error" is marshaled as nil by the standard json library.
//
// If the obj implements custom JSON marshalling or is not an error, the obj is returned unchanged.
//
// The boolean return value is true if the obj was converted, false otherwise.
func convertForJSONMarshalling(obj interface{}) (interface{}, bool) {
	switch t := obj.(type) {
	case json.Marshaler,
		encoding.TextMarshaler,
		*Error,
		*ErrorList:
		// no conversion needed - they marshal correctly
	case error:
		return t.Error(), true
	}
	return obj, false
}

// isNilValue reports whether err is a non-nil error interface wrapping a nil pointer (or other nilable value), e.g. a
// nil *Error returned from a function and passed on as-is. Such a value is != nil when compared as an interface, but
// dereferencing it later (e.g. in effectiveKind()) would panic. Rejecting it here keeps a nil cause from ever being
// stored.
func isNilValue(err error) bool {
	v := reflect.ValueOf(err)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

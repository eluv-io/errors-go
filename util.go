package errors

import (
	"bytes"
	"encoding"
	"encoding/json"
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

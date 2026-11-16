package errors_test

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eluv-io/errors-go"
)

func TestErrorArray_JSON(t *testing.T) {
	// the array is marshalled as a value in an interface, where a pointer MarshalJSON would be skipped
	type status struct {
		Errors errors.ErrorArray `json:"errors"`
	}
	arr := errors.ErrorArray{}
	arr.Append(
		errors.NoTrace("read", errors.K.IO, io.EOF),
		&errors.ErrorList{Errors: []error{io.ErrClosedPipe, io.ErrUnexpectedEOF}},
	)
	require.Len(t, arr.Errors, 3, "Append must flatten a nested list")

	var custom any = status{Errors: arr}
	bts, err := json.Marshal(custom)
	require.NoError(t, err)
	require.Equal(t,
		`{"errors":[{"op":"read","kind":"I/O error","cause":"EOF"},"io: read/write on closed pipe","unexpected EOF"]}`,
		string(bts))

	var unmarshalled status
	require.NoError(t, json.Unmarshal(bts, &unmarshalled))
	require.Equal(t, arr.Error(), unmarshalled.Errors.Error())

	custom = status{}
	bts, err = json.Marshal(custom)
	require.NoError(t, err)
	require.Equal(t, `{"errors":null}`, string(bts))

	unmarshalled = status{}
	require.NoError(t, json.Unmarshal(bts, &unmarshalled))
	require.Empty(t, unmarshalled.Errors.Errors)
}

func TestErrorArray_JSONNestedCause(t *testing.T) {
	arr := errors.ErrorArray{}
	arr.Append(errors.NoTrace("copy", errors.K.Invalid, errors.Append(
		errors.NoTrace("read", errors.K.Permission, "file", "a.txt",
			errors.Append(errors.NoTrace("open", errors.K.NotExist), io.ErrUnexpectedEOF)),
		io.EOF,
	)))

	bts, err := json.Marshal(arr)
	require.NoError(t, err)
	require.Equal(t,
		`[{"op":"copy","kind":"invalid","cause":[{"op":"read","kind":"permission denied","file":"a.txt",`+
			`"cause":[{"op":"open","kind":"item does not exist"},"unexpected EOF"]},"EOF"]}]`,
		string(bts))

	var unmarshalled errors.ErrorArray
	require.NoError(t, json.Unmarshal(bts, &unmarshalled))
	require.Len(t, unmarshalled.Errors, 1)
	cause, ok := errors.Wrap(unmarshalled.Errors[0]).Cause().(*errors.ErrorArray)
	require.True(t, ok, "a nested array must be decoded as an ErrorArray")
	require.Len(t, cause.Errors, 2)
	require.Equal(t, errors.K.Permission, errors.Wrap(cause.Errors[0]).Kind())

	remarshalled, err := json.Marshal(unmarshalled)
	require.NoError(t, err)
	require.Equal(t, string(bts), string(remarshalled))

	// an empty array is no cause
	var e errors.Error
	require.NoError(t, json.Unmarshal([]byte(`{"op":"copy","cause":[]}`), &e))
	require.Nil(t, e.Cause())
}

func TestErrorArray_Append(t *testing.T) {
	list := func() error {
		return errors.Append(errors.NoTrace("read", errors.K.IO), io.EOF)
	}
	nested := errors.NoTrace("copy", errors.K.Invalid, list())

	arr := errors.ErrorArray{}
	arr.Append(nested, list(), nil, (*errors.ErrorList)(nil))
	require.Len(t, arr.Errors, 3, "Append must flatten lists and skip nil")

	bts, err := json.Marshal(arr)
	require.NoError(t, err)
	require.Equal(t,
		`[{"op":"copy","kind":"invalid","cause":[{"op":"read","kind":"I/O error"},"EOF"]},`+
			`{"op":"read","kind":"I/O error"},"EOF"]`,
		string(bts))

	bts, err = json.Marshal(nested)
	require.NoError(t, err)
	require.Contains(t, string(bts), `"errors":`, "Append must not modify the appended errors")
	require.Equal(t,
		`{"op":"copy","kind":"invalid","cause":{"errors":[{"op":"read","kind":"I/O error"},"EOF"]}}`,
		string(bts))

	// errors set directly are marshalled as they are
	bts, err = json.Marshal(errors.ErrorArray{ErrorList: errors.ErrorList{Errors: []error{nested}}})
	require.NoError(t, err)
	require.Contains(t, string(bts), `"errors":`)
	require.Equal(t,
		`[{"op":"copy","kind":"invalid","cause":{"errors":[{"op":"read","kind":"I/O error"},"EOF"]}}]`,
		string(bts))
}

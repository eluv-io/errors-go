package errors_test

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eluv-io/errors-go"
)

func TestErrorListBasic(t *testing.T) {
	var err error

	err = errors.Append(err, nil)
	require.Nil(t, err)

	err = errors.Append(err, nil, nil)
	require.Nil(t, err)

	err = errors.Append(err, io.EOF)
	require.Equal(t, io.EOF, err)

	err = errors.Append(err, nil, nil)
	require.Equal(t, io.EOF, err)

	err = errors.Append(err, io.ErrUnexpectedEOF)
	require.Equal(t, &errors.ErrorList{Errors: []error{io.EOF, io.ErrUnexpectedEOF}}, err)
}

func TestAppend(t *testing.T) {
	require.Nil(t, errors.Append(nil))
	require.Nil(t, errors.Append(nil, nil))
	require.Equal(t, io.EOF, errors.Append(io.EOF))
	require.Equal(t, io.EOF, errors.Append(nil, io.EOF))
	assertErrorList(t,
		errors.Append(io.EOF, io.ErrClosedPipe),
		io.EOF, io.ErrClosedPipe)
	assertErrorList(t,
		errors.Append(nil, io.EOF, io.ErrClosedPipe),
		io.EOF, io.ErrClosedPipe)
	assertErrorList(
		t,
		errors.Append(io.EOF, errors.Append(io.ErrClosedPipe, io.ErrNoProgress)),
		io.EOF, io.ErrClosedPipe, io.ErrNoProgress,
	)

	var list *errors.ErrorList
	require.Nil(t, errors.Append(list))
	require.Nil(t, errors.Append(list, nil))
	require.Equal(t, io.EOF, errors.Append(list, io.EOF))

	list = nil
	assertErrorList(t, errors.Append(list, io.EOF, io.ErrUnexpectedEOF), io.EOF, io.ErrUnexpectedEOF)
}

func TestErrorList_Append(t *testing.T) {
	var el errors.ErrorList

	el.Append(nil)
	require.Len(t, el.Errors, 0)
	require.Nil(t, el.ErrorOrNil())

	el.Append(&errors.ErrorList{})
	require.Len(t, el.Errors, 0)
	require.Nil(t, el.ErrorOrNil())

	el.Append(io.EOF)
	require.Len(t, el.Errors, 1)
	require.NotNil(t, el.ErrorOrNil())
	require.Equal(t, io.EOF, el.ErrorOrNil())

	el.Append(io.ErrUnexpectedEOF)
	require.Len(t, el.Errors, 2)
	require.NotNil(t, el.ErrorOrNil())
	require.Equal(t, &el, el.ErrorOrNil())
}

func TestErrorList_Error(t *testing.T) {
	el := &errors.ErrorList{}
	require.Equal(t, "", el.Error())

	el.Append(io.EOF)
	require.Equal(t, "EOF", el.Error())
}

func TestErrorList_MarshalJSON(t *testing.T) {
	var list error
	list = errors.Append(errors.E("read", errors.K.IO, io.EOF), io.ErrClosedPipe)
	bts, err := json.MarshalIndent(errors.E("wrapped", list), "", "  ")
	require.NoError(t, err)
	fmt.Println(string(bts))

	var unmarshalled interface{}
	err = json.Unmarshal(bts, &unmarshalled)
	require.NoError(t, err)

	type msi = map[string]interface{}

	errs := unmarshalled.(msi)["cause"].(msi)["errors"].([]interface{})

	require.Equal(t, "read", errs[0].(msi)["op"])
	require.Equal(t, string(errors.K.IO), errs[0].(msi)["kind"])
	require.Equal(t, io.ErrClosedPipe.Error(), errs[1])
}

func TestErrorList_UnmarshalJSON(t *testing.T) {
	var err error
	list := errors.ErrorList{}

	err = list.UnmarshalJSON(nil)
	assert.Error(t, err)

	err = list.UnmarshalJSON([]byte("invalid JSON::"))
	assert.Error(t, err)

	err = list.UnmarshalJSON([]byte(`{}`))
	assert.NoError(t, err)

	err = list.UnmarshalJSON([]byte(`{"blub":"blob"}`))
	assert.NoError(t, err)

	err = list.UnmarshalJSON([]byte(`{"errors":["EOF",{"op":"op1","kind":"invalid"}]}`))
	require.NoError(t, err)
	assert.Equal(t, errors.Str("EOF"), list.Errors[0])
	assert.Equal(t, errors.NoTrace("op1", errors.K.Invalid).Error(), list.Errors[1].Error())
}

func TestErrorList_JSON(t *testing.T) {
	createList := func(errs ...error) error {
		list := errors.ErrorList{}
		list.Append(errs...)
		return &list
	}
	eof := errors.Str("EOF")
	oom := errors.Str("OOM")
	tests := []struct {
		list error
	}{
		{createList()},
		{createList(eof)},
		{createList(eof, oom)},
		{createList(errors.E("read", errors.K.IO))},
		{createList(errors.E("read", errors.K.IO, eof), oom)},
	}

	for idx, test := range tests {
		t.Run(fmt.Sprint(idx), func(t *testing.T) {
			bts, err := json.MarshalIndent(test.list, "", "  ")
			require.NoError(t, err)
			fmt.Println(string(bts))

			var unmarshalled errors.ErrorList
			err = json.Unmarshal(bts, &unmarshalled)
			require.NoError(t, err)

			assert.Equal(t, test.list.Error(), unmarshalled.Error())
		})
	}
}

func ExampleAppend() {
	{
		fmt.Println("nil error:")
		fmt.Println(errors.Append(nil, nil))
		fmt.Println()
	}

	{
		var err error
		err = errors.Append(err, io.EOF)
		fmt.Println("single error:")
		fmt.Println(err)
		fmt.Println()
	}

	{
		var err error
		err = errors.Append(err, io.EOF)
		err = errors.Append(err, io.ErrNoProgress)
		err = errors.Append(err, io.ErrClosedPipe)
		fmt.Println("multiple errors:")
		fmt.Println(err)
	}

	{
		var err error
		err = errors.Append(errors.E("read", errors.K.IO, io.EOF), errors.E("write", errors.K.Cancelled))
		fmt.Println("complex errors:")
		fmt.Println(err)

		/*
			With stacktraces enabled it would look like this:
			error-list count [2]
				0: op [read] kind [I/O error] cause [EOF]
				github.com/eluv-io/errors-go/list_test.go:192 ExampleAppend()
				testing/run_example.go:64                     runExample()
				testing/example.go:44                         runExamples()
				testing/testing.go:1505                       (*M).Run()
				_testmain.go:165                              main()

				1: op [write] kind [operation cancelled]
				github.com/eluv-io/errors-go/list_test.go:192 ExampleAppend()
				testing/run_example.go:64                     runExample()
				testing/example.go:44                         runExamples()
				testing/testing.go:1505                       (*M).Run()
				_testmain.go:165                              main()
		*/
	}

	// Output:
	//
	// nil error:
	// <nil>
	//
	// single error:
	// EOF
	//
	// multiple errors:
	// error-list count [3]
	//	0: EOF
	//	1: multiple Read calls return no data or error
	//	2: io: read/write on closed pipe
	//
	// complex errors:
	// error-list count [2]
	//	0: op [read] kind [I/O error] cause [EOF]
	//	1: op [write] kind [operation cancelled]
	//
}

func assertErrorList(t *testing.T, list error, expected ...error) {
	switch list := list.(type) {
	case *errors.ErrorList:
		if list == nil {
			require.Nil(t, expected)
		} else {
			require.Equal(t, expected, list.Errors)
		}
	default:
		require.Fail(t, "not an error list", list)
	}
}

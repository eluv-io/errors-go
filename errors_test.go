package errors_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eluv-io/errors-go"
)

func init() {
	errors.PrintStacktrace = false
	errors.MarshalStacktraceAsArray = false
}

func TestE(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		err := errors.E()
		assert.Equal(t, errors.K.Other, err.Field("kind"))
		assert.Equal(t, "kind [unclassified error]", err.Error())
	})
	t.Run("one arg", func(t *testing.T) {
		err := errors.E("operation")
		assert.Equal(t, "operation", err.Field("op"))
		assert.Equal(t, errors.K.Other, err.Field("kind"))
		assert.Equal(t, "op [operation] kind [unclassified error]", err.Error())
	})
	t.Run("two args", func(t *testing.T) {
		err := errors.E("operation", errors.K.IO)
		assert.Equal(t, "operation", err.Field("op"))
		assert.Equal(t, errors.K.IO, err.Field("kind"))
		assert.Equal(t, "op [operation] kind [I/O error]", err.Error())
	})
	t.Run("three args", func(t *testing.T) {
		err := errors.E("operation", errors.K.IO, io.EOF)
		assert.Equal(t, "operation", err.Field("op"))
		assert.Equal(t, errors.K.IO, err.Field("kind"))
		assert.Equal(t, io.EOF, err.Field("cause"))
		assert.Equal(t, "op [operation] kind [I/O error] cause [EOF]", err.Error())
	})
	t.Run("four args", func(t *testing.T) {
		err := errors.E("operation", errors.K.IO, io.EOF, "value without key")
		assert.Equal(t, "operation", err.Field("op"))
		assert.Equal(t, errors.K.IO, err.Field("kind"))
		assert.Equal(t, io.EOF, err.Field("cause"))
		assert.Equal(t, nil, err.Field("key"))
		assert.Equal(t, "op [operation] kind [I/O error] value without key [<missing>] cause [EOF]", err.Error())
	})
	t.Run("five args", func(t *testing.T) {
		err := errors.E("operation", errors.K.IO, io.EOF, "key", "val")
		assert.Equal(t, "operation", err.Field("op"))
		assert.Equal(t, errors.K.IO, err.Field("kind"))
		assert.Equal(t, io.EOF, err.Field("cause"))
		assert.Equal(t, "val", err.Field("key"))
		assert.Equal(t, "op [operation] kind [I/O error] key [val] cause [EOF]", err.Error())
	})
	t.Run("more args", func(t *testing.T) {
		err := errors.E("operation", errors.K.IO, io.EOF, "key1", "val1", "key2", "val2", "key3", "val3")
		assert.Equal(t, "operation", err.Field("op"))
		assert.Equal(t, errors.K.IO, err.Field("kind"))
		assert.Equal(t, io.EOF, err.Field("cause"))
		assert.Equal(t, "val1", err.Field("key1"))
		assert.Equal(t, "val2", err.Field("key2"))
		assert.Equal(t, "val3", err.Field("key3"))
		assert.Equal(t, "op [operation] kind [I/O error] key1 [val1] key2 [val2] key3 [val3] cause [EOF]", err.Error())
	})
	t.Run("one arg slice", func(t *testing.T) {
		fields := []interface{}{"operation", errors.K.IO, io.EOF, "key1", "val1", "key2", "val2", "key3", "val3"}
		err := errors.E(fields)
		assert.Equal(t, "op [operation] kind [I/O error] key1 [val1] key2 [val2] key3 [val3] cause [EOF]", err.Error())
	})
}

func TestE_invalidArgs(t *testing.T) {
	err := errors.E(fmt.Errorf("%s", "some error"), nil, "arg1", 7, "arg2")
	assert.Equal(t, "some error", err.Field("cause").(error).Error())
	assert.Equal(t, errors.K.Other, err.Field("kind"))
	assert.Equal(t, 7, err.Field("arg1"))
	assert.Equal(t, "<missing>", err.Field("arg2"))
	assert.Equal(t, "kind [unclassified error] arg1 [7] arg2 [<missing>] cause [some error]", err.Error())
}

func TestE_nilArgs(t *testing.T) {
	err := errors.E(nil, nil, nil, nil)
	assert.Equal(t, "kind [unclassified error]", err.Error())
}

func TestWith(t *testing.T) {
	fields := []interface{}{"key1", "val1", "key2", "val2", "key3", "val3"}
	for i := 0; i < 2; i++ {
		var err *errors.Error
		if i == 0 {
			err = errors.E("operation", errors.K.IO, io.EOF).With(fields...)
		} else {
			err = errors.E("operation", errors.K.IO, io.EOF).With(fields)
		}
		assert.Equal(t, "operation", err.Field("op"))
		assert.Equal(t, errors.K.IO, err.Field("kind"))
		assert.Equal(t, io.EOF, err.Field("cause"))
		assert.Equal(t, "val1", err.Field("key1"))
		assert.Equal(t, "val2", err.Field("key2"))
		assert.Equal(t, "val3", err.Field("key3"))
		assert.Equal(t, "op [operation] kind [I/O error] key1 [val1] key2 [val2] key3 [val3] cause [EOF]", err.Error())
	}

	// test overriding op/kind/cause
	assert.Equal(t, "op2", errors.E("op1").With("op", "op2").Op())
	assert.Equal(t, errors.K.Invalid, errors.E(errors.K.IO).With("kind", errors.K.Invalid).Kind())
	assert.Equal(t, errors.K.Invalid, errors.E(errors.K.IO).With("kind", string(errors.K.Invalid)).Kind())
	assert.Equal(t, io.EOF, errors.E(io.ErrUnexpectedEOF).With("cause", io.EOF).Cause())
}

func TestEquivalence(t *testing.T) {
	eq := func(err1, err2 error) {
		assert.Equal(t, err1.Error(), err2.Error())
	}
	eq(
		errors.E("operation", errors.K.IO, io.EOF, "key1", "val1", "key2", "val2", "key3", "val3"),
		errors.E().WithOp("operation").WithKind(errors.K.IO).WithCause(io.EOF).With("key1", "val1").With("key2", "val2").With("key3", "val3"))
	eq(
		errors.E("operation", errors.K.IO, io.EOF, "key1", "val1", "key2", "val2", "key3", "val3"),
		errors.E("operation", errors.K.IO, io.EOF).With("key1", "val1", "key2", "val2", "key3", "val3"))
	eq(
		errors.E("operation", errors.K.IO, io.EOF, "key1", "val1", "key2", "val2", "key3", "val3"),
		errors.E("operation", errors.K.IO, io.EOF, "key1", "val1").With("key2", "val2", "key3", "val3"))
	eq(
		errors.E("operation", errors.K.IO, "key1", "val1", "single_val"),
		errors.E("operation", errors.K.IO).With("key1", "val1", "single_val"))
	eq(
		errors.E("operation", errors.K.IO, "key1", "val1", "key2"),
		errors.Str("op [operation] kind [I/O error] key1 [val1] key2 [<missing>]"))

	nested := errors.E("nested", errors.K.NotExist)
	eq(
		errors.E().WithOp("operation").WithCause(nested).With("key1", "val1"),
		errors.E("operation", nested, "key1", "val1"))
}

func TestIsKind(t *testing.T) {
	// kind defaults to Other
	assert.True(t, errors.IsKind(errors.K.Other, errors.E()))

	assert.True(t, errors.IsKind(errors.K.IO, errors.E("op", errors.K.IO, io.EOF)))
	assert.True(t, errors.IsKind(errors.K.NotExist, errors.E("op", errors.E("op_nested", errors.K.NotExist))))
	assert.True(t, errors.IsKind(errors.K.NotExist, errors.E("op1", errors.K.Invalid, errors.E("op2", errors.K.NotExist))))
	assert.True(t, errors.IsKind(errors.K.NotExist, errors.E("op1", errors.K.Invalid, errors.E("op2", errors.K.Invalid, errors.E("op3", errors.K.Invalid, errors.E("op4", errors.K.NotExist))))))

	assert.False(t, errors.IsKind(errors.K.IO, errors.E("op", errors.K.NotExist, io.EOF)))
	assert.False(t, errors.IsKind(errors.K.NotExist, errors.E("op", errors.K.Invalid, errors.E("op_nested", errors.K.Other))))
}

func TestIsNotExist(t *testing.T) {
	assert.False(t, errors.IsNotExist(errors.E("op", errors.K.IO, io.EOF)))
	assert.True(t, errors.IsNotExist(errors.E("op", errors.E("op_nested", errors.K.NotExist))))
	assert.True(t, errors.IsNotExist(errors.E("op1", errors.K.Invalid, errors.E("op2", errors.K.NotExist))))
	assert.True(t, errors.IsNotExist(errors.E("op1", errors.K.Invalid, errors.E("op2", errors.K.Invalid, errors.E("op3", errors.K.Invalid, errors.E("op4", errors.K.NotExist))))))
}

const (
	op  = "Op"
	op1 = "Op1"
	op2 = "Op2"
)

func TestMatch(t *testing.T) {
	eof := errors.E(op, errors.K.Invalid, io.EOF, "k1", "v1", "k2", "v2")
	errConnect := errors.E("connect", errors.K.IO, errors.Str("network unreachable"), "k1", "v1", "k2", "v2")
	errSendEmail := errors.E("send email", errConnect)

	type matchTest struct {
		err1, err2 error
		matched    bool
	}
	matchTests := []matchTest{
		// Errors not of type *Error are compared with reflect.DeepEqual
		{
			nil,
			nil,
			true,
		},
		{
			io.EOF,
			io.EOF,
			true,
		},
		{
			errors.E(io.EOF),
			io.EOF,
			false,
		},
		{
			errors.E(op, io.EOF),
			io.EOF,
			false,
		},
		{
			io.EOF,
			errors.E(io.EOF),
			true,
		},
		// Success. We can drop fields from the first argument and still match.
		{
			errors.E(op, errors.K.Invalid, io.EOF, "k1", "v1", "k2", "v2"),
			eof,
			true,
		},
		{
			errors.E(op, errors.K.Invalid, io.EOF, "k1", "v1"),
			eof,
			true,
		},
		{
			errors.E(op, errors.K.Invalid, io.EOF),
			eof,
			true,
		},
		{
			errors.E(op, errors.K.Invalid),
			eof,
			true,
		},
		{
			errors.E(op),
			eof,
			true,
		},
		{
			errors.E(op),
			eof,
			true,
		},
		// Failure.
		{
			errors.E(io.EOF),
			errors.E(io.ErrClosedPipe),
			false,
		},
		{
			errors.E(op1),
			errors.E(op2),
			false,
		},
		{
			errors.E(errors.K.Invalid),
			errors.E(errors.K.Permission),
			false,
		},
		{
			eof,
			errors.E(op, errors.K.Permission, io.EOF),
			false,
		},
		{
			errors.E(op1, errors.Str("something")),
			errors.E(op1),
			false,
		},
		// Nested *Errors.
		{
			errors.E(op1, errors.E(op2)),
			errors.E(op1, errors.E(op2)),
			true,
		},
		{
			errors.E(op1),
			errors.E(op1, errors.E(op2)),
			true,
		},
		{
			errors.E(op1, errors.E(op)),
			errors.E(errors.E(op2).Error(), op1),
			false,
		},
		{
			errors.E().With("k1", "v1"),
			errConnect,
			true,
		},
		{
			errors.E().With("k1", "another value"),
			errConnect,
			false,
		},
		{
			errors.E().With("missing", "value"),
			errConnect,
			false,
		},
		{
			errors.E(errors.K.IO, "k2", "v2"),
			errConnect,
			true,
		},
		{
			errConnect,
			errSendEmail,
			false,
		},
		{
			errors.E(errConnect),
			errSendEmail,
			true,
		},
		{
			errors.E(errors.E(errors.K.IO, "k2", "v2")),
			errSendEmail,
			true,
		},
		{
			errors.E("finish signup", "nested", errConnect),
			errors.E("finish signup", "nested", errConnect),
			true,
		},
		{
			errors.E("finish signup", "nested", errConnect),
			errors.E("finish signup", "nested", "not an error"),
			false,
		},
		{
			errors.E("finish signup", "nested", errConnect),
			errors.E("finish signup"),
			false,
		},
	}

	for idx, test := range matchTests {
		matched := errors.Match(test.err1, test.err2)
		assert.Equal(t, test.matched, matched, "#%d err1 [%q] err2 [%q]", idx, test.err1, test.err2)
	}
}

func TestSeparator(t *testing.T) {
	defer func(prev string) {
		errors.Separator = prev
	}(errors.Separator)
	errors.Separator = ":: "

	err := createNestedError()

	want := "op [send email] kind [I/O error] cause:: op [connect] kind [I/O error] k1 [v1] cause [network unreachable]"
	assert.Equal(t, want, err.Error())
}

func TestGetField(t *testing.T) {
	e1 := errors.E("Test", "key", "val1")
	e2 := errors.E("Test", e1, "key", "val2")

	f, ok := errors.GetField(nil, "key")
	require.False(t, ok)
	require.Equal(t, f, "")

	f, ok = errors.GetField(io.EOF, "key")
	require.False(t, ok)
	require.Equal(t, f, "")

	f, ok = errors.GetField(e2, "key")
	require.True(t, ok)
	require.Equal(t, "val2", f)

	e2 = errors.E("Test", e1, "another_key", "val2")
	f, ok = errors.GetField(e2, "key")
	require.True(t, ok)
	require.Equal(t, "val1", f)

	e3 := errors.E("Test", e2, "yet_another_key", "val3")
	f, ok = errors.GetField(e3, "key")
	require.True(t, ok)
	require.Equal(t, "val1", f)

	e2 = errors.E("Test", e1, "key", "val2")
	e3 = errors.E("Test", e2, "yet_another_key", "val3")
	f, ok = errors.GetField(e3, "key")
	require.True(t, ok)
	require.Equal(t, "val2", f)

	fe1 := fmt.Errorf("not an elv error %s", "x")
	e2 = errors.E("Test", fe1, "another_key", "val2")
	e3 = errors.E("Test", e2, "yet_another_key", "val3")
	f, ok = errors.GetField(e3, "key")
	require.False(t, ok)

}

func TestField(t *testing.T) {
	e1 := errors.E("Test", "key", "val1")
	e2 := errors.E("Test", e1, "key", 2)

	f := errors.Field(nil, "key")
	require.Nil(t, f)

	f = errors.Field(io.EOF, "key")
	require.Nil(t, f)

	f = errors.Field(e1, "missing_key")
	require.Nil(t, f)

	f = errors.Field(e2, "key")
	require.Equal(t, 2, f)

	e2 = errors.E("Test", e1, "another_key", "val2")
	f = errors.Field(e2, "key")
	require.Equal(t, "val1", f)

	e3 := errors.E("Test", e2, "yet_another_key", 3)
	f = errors.Field(e3, "key")
	require.Equal(t, "val1", f)
	f = errors.Field(e3, "yet_another_key")
	require.Equal(t, 3, f)

	e2 = errors.E("Test", e1, "key", 2)
	e3 = errors.E("Test", e2, "yet_another_key", "val3")
	f = errors.Field(e3, "key")
	require.Equal(t, 2, f)

	fe1 := fmt.Errorf("not an elv error %s", "x")
	e2 = errors.E("Test", fe1, "another_key", "val2")
	e3 = errors.E("Test", e2, "yet_another_key", "val3")
	f = errors.Field(e3, "key")
	require.Nil(t, f)
}

func TestGetRoot(t *testing.T) {
	var e interface{}
	require.Nil(t, errors.GetRoot(e))

	e = errors.E("test")
	require.Equal(t, e, errors.GetRoot(e))

	e = errors.E("test", io.EOF)
	require.Equal(t, e, errors.GetRoot(e))

	root := errors.E("root")

	e = errors.E("test", root)
	require.Equal(t, root, errors.GetRoot(e))

	e = errors.E("test", errors.E("cause1", errors.E("cause2", errors.E("cause3", root))))
	require.Equal(t, root, errors.GetRoot(e))
}

func TestGetRootCause(t *testing.T) {
	var e error
	require.Equal(t, errors.NilError, errors.GetRootCause(e))

	e = errors.E("test")
	require.Equal(t, errors.NilError, errors.GetRootCause(e))

	e = errors.E("test", io.EOF)
	require.Equal(t, io.EOF, errors.GetRootCause(e))

	root := errors.E("root")

	e = errors.E("test", root)
	require.Equal(t, errors.NilError, errors.GetRootCause(e))

	e = errors.E("test", errors.E("cause1", errors.E("cause2", errors.E("cause3", root))))
	require.Equal(t, errors.NilError, errors.GetRootCause(e))

	root = errors.E("root", io.EOF)

	e = errors.E("test", root)
	require.Equal(t, io.EOF, errors.GetRootCause(e))

	e = errors.E("test", errors.E("cause1", errors.E("cause2", errors.E("cause3", root))))
	require.Equal(t, io.EOF, errors.GetRootCause(e))

	// cause is converted to std error if it doesn't implement error interface
	e = errors.E("test", "cause", "not an error")
	require.Equal(t, errors.Str("not an error"), errors.GetRootCause(e))

	// nil cause is ignored
	e = errors.E("test", "cause", nil)
	require.Equal(t, errors.NilError, errors.GetRootCause(e))
}

func TestWrap(t *testing.T) {
	assert.Nil(t, errors.Wrap(nil))
	assert.Nil(t, errors.Wrap(nil, "key", "value"))

	assert.Equal(t, "kind [unclassified error] cause [EOF]", errors.Wrap(io.EOF).Error())
	assert.Equal(t, "kind [unclassified error] key [val] cause [EOF]", errors.Wrap(io.EOF, "key", "val").Error())

	err := errors.E("read", errors.K.Invalid, "cause", "bad weather")
	assert.Equal(t, err, errors.Wrap(err))
	assert.Equal(t, "op [read] kind [invalid] key [val] cause [bad weather]", errors.Wrap(err, "key", "val").Error())
}

func TestIgnore(t *testing.T) {
	errors.Ignore(nil) // ensure no crash

	called := 0
	errors.Ignore(func() error {
		called++
		return nil
	})
	assert.Equal(t, 1, called)

	errors.Ignore(func() error {
		called++
		return io.EOF
	})
	assert.Equal(t, 2, called)
}

func TestLog(t *testing.T) {
	type logMsg struct {
		msg    string
		fields []interface{}
	}

	var lastMsg *logMsg
	logFn := func(msg string, fields ...interface{}) {
		lastMsg = &logMsg{
			msg:    msg,
			fields: fields,
		}
	}

	errors.Log(nil, logFn) // ensure no crash
	assert.Nil(t, lastMsg)

	called := 0
	errors.Log(func() error {
		called++
		return nil
	}, logFn)
	assert.Equal(t, 1, called)
	assert.Nil(t, lastMsg)

	errors.Log(func() error {
		called++
		return io.EOF
	}, logFn)
	assert.Equal(t, 2, called)
	assert.NotNil(t, lastMsg)
	assert.Equal(t, "errors.Log function call returned error", lastMsg.msg)

	lastMsg = nil
	errors.Log(func() error {
		called++
		return io.EOF
	}, nil)
	assert.Equal(t, 3, called)
	assert.Nil(t, lastMsg)
}

func TestFromContext(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, errors.FromContext(nil))
	})

	t.Run("context cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		assert.Nil(t, errors.FromContext(ctx))
		cancel()
		assert.NotNil(t, errors.FromContext(ctx))
	})

	t.Run("context deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		assert.Nil(t, errors.FromContext(ctx))
		time.Sleep(100 * time.Millisecond)
		assert.NotNil(t, errors.FromContext(ctx))
		cancel()
		assert.NotNil(t, errors.FromContext(ctx))
	})

	t.Run("custom context error", func(t *testing.T) {
		ctx := new(customContext)
		assert.Equal(t, io.EOF, errors.FromContext(ctx).Cause())
	})

}

func TestTypeOf(t *testing.T) {
	assert.Equal(t, "<nil>", errors.TypeOf(nil))
	assert.Equal(t, "int", errors.TypeOf(0))
	assert.Equal(t, "int64", errors.TypeOf(int64(0)))
}

func TestError_Kind(t *testing.T) {
	tests := []struct {
		want interface{}
		err  *errors.Error
	}{
		{errors.K.Other, errors.E()},
		{errors.K.Invalid, errors.E(errors.K.Invalid)},
		{errors.K.Invalid, errors.E().WithKind(errors.K.Invalid)},
	}
	for idx, tt := range tests {
		t.Run(fmt.Sprint(idx, tt.want), func(t *testing.T) {
			require.Equal(t, tt.want, tt.err.Kind())

		})
	}
}

func TestError_OpKindCauseArgs(t *testing.T) {
	errs := []*errors.Error{
		errors.E().WithOp("operation").WithKind(errors.K.IO).WithCause(io.EOF),
		errors.E("operation", errors.K.IO, io.EOF),
		errors.E("operation", "kind", errors.K.IO, "cause", io.EOF),
	}
	for _, err := range errs {
		assert.Equal(t, "op [operation] kind [I/O error] cause [EOF]", err.Error())

		assert.Equal(t, "operation", err.Op())
		assert.Equal(t, errors.K.IO, err.Kind())
		assert.Equal(t, io.EOF, err.Cause())

		assert.Equal(t, "operation", err.Field("op"))
		assert.Equal(t, errors.K.IO, err.Field("kind"))
		assert.Equal(t, io.EOF, err.Field("cause"))
	}
}

func TestError_WithCause(t *testing.T) {
	tests := []struct {
		name string
		err  *errors.Error
		want string
	}{
		{"std error", errors.E().WithCause(io.EOF), "kind [unclassified error] cause [EOF]"},
		{"nil", errors.E().WithCause(nil), "kind [unclassified error]"},
		{"*Error: override kind", errors.E().WithCause(errors.E(errors.K.Invalid)), "kind [invalid] cause:\n\tkind [invalid]"},
		{"*Error: do not override uninitialized kind", errors.E(errors.K.IO).WithCause(errors.E(errors.K.Invalid)), "kind [I/O error] cause:\n\tkind [invalid]"},
		{"*Error: do not override kind initialized to K.Other", errors.E(errors.K.Other).WithCause(errors.E(errors.K.Invalid)), "kind [unclassified error] cause:\n\tkind [invalid]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, test.err.Error())
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	var err error
	err = createNestedError()
	fmt.Println(err)
	require.Contains(t, err.Error(), "send email")

	err = err.(*errors.Error).Unwrap()
	require.Error(t, err)
	require.Contains(t, err.Error(), "connect")

	err = err.(*errors.Error).Unwrap()
	require.Error(t, err)
	require.Equal(t, err.Error(), "network unreachable")

	require.Nil(t, errors.E("noop").Unwrap())
}

func TestError_FormatError(t *testing.T) {
	var err *errors.Error
	assert.Equal(t, "", err.Error())
	assert.Equal(t, "", err.FormatError(true))

	err = errors.E("op", errors.K.Invalid, io.EOF, "k1", "v1", "k2", "v2", "k3", "v3")
	assert.Equal(t, "op [op] kind [invalid] k1 [v1] k2 [v2] k3 [v3] cause [EOF]", err.Error())
	assert.Equal(t, "op [op] kind [invalid] k1 [v1] k2 [v2] k3 [v3] cause [EOF]", err.FormatError(false, errors.DefaultFieldOrder...))
	assert.Equal(t, "op [op] kind [invalid] k2 [v2] k1 [v1] k3 [v3] cause [EOF]", err.FormatError(false, "op", "kind", "k2", "", "cause"))
	assert.Equal(t, "k3 [v3] k2 [v2] op [op] kind [invalid] k1 [v1] cause [EOF]", err.FormatError(false, "k3", "k2"))
}

func TestError_MarshalJSON(t *testing.T) {
	revert := enableStacktraces()
	defer revert()

	fo := errors.DefaultFieldOrder
	defer func() {
		errors.DefaultFieldOrder = fo
	}()

	errs := []*errors.Error{
		errors.E("noop"),
		errors.E("noop").WithCause(nil),
		errors.E("noop").WithCause(io.EOF),
		createNestedError(),
		errors.E("op", errors.K.Invalid, io.EOF, "k1", "v1", "k2", "v2", "k3", "v3"),
	}

	for _, fieldOrder := range [][]string{errors.DefaultFieldOrder, {"kind", "op", "", "cause"}} {
		t.Run(fmt.Sprint("field order", fieldOrder), func(t *testing.T) {
			errors.DefaultFieldOrder = fieldOrder
			for _, e1 := range errs {
				t.Run(e1.Error(), func(t *testing.T) {
					b, err := json.MarshalIndent(e1, "", "  ")
					assert.NoError(t, err)

					jsn := string(b)
					fmt.Println(jsn)

					// assert field order
					idx := -1
					for _, field := range fieldOrder {
						got := strings.Index(jsn, `"`+field+`"`)
						if got != -1 {
							assert.Less(t, idx, got, field)
							idx = got
						}
					}

					// assert stacktrace exists on top-level error
					var m map[string]interface{}
					err = json.Unmarshal(b, &m)
					require.NoError(t, err)
					require.NotEmpty(t, m["stacktrace"], m)
					ok := true
					for ok {
						// ... but not on nested errors
						if m, ok = m["cause"].(map[string]interface{}); ok {
							require.Nil(t, m["stacktrace"])
						}
					}

					var e2 = &errors.Error{}
					err = json.Unmarshal(b, e2)
					require.NoError(t, err)

					fmt.Println("unmarshalled:", e2)

					// can't compare because causes of type error (vs. *Error) are not the
					// original errors...
					// require.Equal(t, e1, e2)

					// therefore, make sure that at least the string representation - without
					// stacktrace - is equal.

					require.Equal(t, e1.ErrorNoTrace(), e2.ErrorNoTrace())
				})
			}
		})
	}
}

func TestError_MarshalJSON_StackAsArray(t *testing.T) {
	revert := enableMarshalStacktraceAsArray()
	defer revert()
	TestError_MarshalJSON(t)
}

func TestTemplate(t *testing.T) {
	for _, template := range []func(fields ...interface{}) errors.TemplateFn{errors.Template, errors.T} {
		fn := func(cause error) error {
			e := template("test")
			return e.IfNotNil(cause)
		}
		if fn(nil) != nil {
			assert.Fail(t, "non-nil!")
		}
		assert.NoError(t, fn(nil))
		assert.Nil(t, fn(nil))
		assert.Equal(t, nil, fn(nil))
		assert.Error(t, fn(io.EOF))

		e := template("test", "key", "value")
		assert.Equal(t,
			errors.E("test", io.EOF, "key", "value").ErrorNoTrace(),
			e.IfNotNil(io.EOF).(*errors.Error).ErrorNoTrace())
		assert.Equal(t,
			errors.E("test", errors.K.IO, io.EOF, "key", "value", "key2", "value2").ErrorNoTrace(),
			e.IfNotNil(io.EOF, "key2", "value2", errors.K.IO).(*errors.Error).ErrorNoTrace())
		assert.Equal(t,
			nil,
			e.IfNotNil(nil))
	}
}

func TestTemplateNoTrace(t *testing.T) {
	revert := enableStacktraces()
	defer revert()

	for _, template := range []func(fields ...interface{}) errors.TemplateFn{errors.TemplateNoTrace, errors.TNoTrace} {
		e := template("test", "key", "value")
		assert.Equal(t,
			"op [test] kind [unclassified error] key [value] cause [EOF]",
			e.IfNotNil(io.EOF).(*errors.Error).Error())
	}
}

func TestTemplateFn_IfNotNil(t *testing.T) {
	e := errors.TemplateNoTrace("op1", errors.K.Invalid)
	require.Equal(t, nil, e.IfNotNil(nil))
	require.Equal(t, e(io.EOF), e.IfNotNil(io.EOF))
}

func TestTemplateFn_Add(t *testing.T) {
	e := errors.Template("op", errors.K.IO, "k1", "v1")
	e = e.Add("k1", "v1.override", "k2", "v2", errors.K.Invalid)
	err := e("k3", "v3", io.EOF)

	assert.Equal(t,
		errors.E("op", "k1", "v1.override", "k2", "v2", "k3", "v3", errors.K.Invalid, io.EOF).ClearStacktrace(),
		err.ClearStacktrace())
}

func TestTemplateFn_Fields(t *testing.T) {
	tests := []struct {
		e    errors.TemplateFn
		want []interface{}
	}{
		{
			errors.Template(),
			nil,
		},
		{
			errors.Template("operation 1", errors.K.Invalid, io.EOF),
			[]interface{}{},
		},
		{
			errors.Template("operation 1", errors.K.Invalid, io.EOF, "k1", "v1"),
			[]interface{}{"k1", "v1"},
		},
		{
			errors.Template("operation 1", "k1", "v1", "k2", "v2"),
			[]interface{}{"k1", "v1", "k2", "v2"},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test.e()), func(t *testing.T) {
			require.Equal(t, test.want, test.e.Fields())
		})
	}
}

func TestTemplateFn_String(t *testing.T) {
	e := errors.TemplateNoTrace("op", errors.K.IO, "k1", "v1")
	require.Equal(t, `op [op] kind [I/O error] k1 [v1]`, e.String())
}

func TestTemplateFn_MarshalJSON(t *testing.T) {
	e := errors.TemplateNoTrace("op", errors.K.IO, "k1", "v1")
	jsn, err := json.Marshal(e)
	require.NoError(t, err)
	require.Equal(t, `{"op":"op","kind":"I/O error","k1":"v1"}`, string(jsn))
}

func TestDefaultKind(t *testing.T) {
	tests := []struct {
		msg  string
		want errors.Kind
		err  *errors.Error
	}{
		{"kind is other if none set", errors.K.Other, errors.E()},
		{"kind is other if none set, even in cause", errors.K.Other, errors.E(errors.E())},
		{"default is used if none set", errors.K.Invalid, errors.E(errors.K.Invalid.Default())},
		{"default in parent is used if none set in cause", errors.K.Invalid, errors.E(errors.K.Invalid.Default(), errors.E())},
		{"kind in cause overrides default in parent", errors.K.Timeout, errors.E(errors.K.Invalid.Default(), errors.E(errors.K.Timeout))},
		{"kind in parent takes precedence over kind in cause", errors.K.Invalid, errors.E(errors.K.Invalid, errors.E(errors.K.NotExist))},
		{"kind in parent takes precedence over default in cause", errors.K.Invalid, errors.E(errors.K.Invalid, errors.E(errors.K.NotExist.Default()))},
		{"default in cause overrides none in parent", errors.K.NotExist, errors.E(errors.E(errors.K.NotExist.Default()))},
		{"default in cause overrides default in parent", errors.K.NotExist, errors.E(errors.K.Invalid.Default(), errors.E(errors.K.NotExist.Default()))},
		{"order of arguments makes no difference", errors.K.NotExist, errors.E(errors.E(errors.K.NotExist.Default()), errors.K.Invalid.Default())},
	}
	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			require.Equal(t, test.want, test.err.Kind())
		})
	}
}

func TestError_UnmarshalJSON(t *testing.T) {
	doPrintStackTrace := false
	v := errors.PrintStacktrace
	errors.PrintStacktrace = doPrintStackTrace
	defer func() { errors.PrintStacktrace = v }()

	e1 := createMoreNestedError()
	b, err := json.Marshal(e1)
	require.NoError(t, err)

	// Note: errors in this test are intentionally created via errors.E()
	// rather than &errors.Error{} since the former also does e.populateStack()
	// and the test verifies the existence of the stack trace
	var e2 = errors.E()

	// except for stack trace verification the test would also work by creating
	// Error like this - note the &
	// var e2 = &errors.Error{}

	err = json.Unmarshal(b, e2)
	require.NoError(t, err)

	se1 := fmt.Sprintf("%v", e1)
	require.Equal(t,
		doPrintStackTrace,
		strings.Index(se1, "goexit()") > 0)

	se2 := fmt.Sprintf("%v", e2)
	// true only if e2 was created via errors.E() and doPrintStackTrace is true
	require.Equal(t,
		doPrintStackTrace,
		strings.Index(se2, "goexit()") > 0)
	// but does not carry stack trace
	require.True(t, strings.Index(se2, "stacktrace") < 0)

	errors.PrintStacktrace = false
	se1 = fmt.Sprintf("%v", e1)
	require.False(t, strings.Index(se1, "runtime.goexit") > 0)

	se2 = fmt.Sprintf("%v", e2)
	require.False(t, strings.Index(se2, "runtime.goexit") > 0)
	require.True(t, strings.Index(se2, "stacktrace") < 0)

	require.Equal(t, se1, se2)

	// remarshal e2
	errors.PrintStacktrace = true
	b, err = json.Marshal(e2)
	require.NoError(t, err)

	var e3 = errors.E()
	err = json.Unmarshal(b, e3)
	require.NoError(t, err)
	se3 := fmt.Sprintf("%v", e3)
	require.True(t, strings.Index(se3, "TestError_UnmarshalJSON()") > 0)
	require.True(t, strings.Index(se3, "stacktrace") < 0)
	require.True(t, e3.Field("stacktrace") == nil)

	fmt.Println("e1")
	fmt.Printf("[%v]\n", e1)
	fmt.Println("\ne2")
	fmt.Printf("[%v]\n", e2)
	fmt.Println("\ne3")
	fmt.Printf("[%v]\n", e3)

	/*
		e1
		[op [send email] kind [unclassified error] cause:
			op [transport] kind [I/O error] cause:
			op [connect] kind [I/O error] cause [network unreachable]]

		e2
		[op [send email] kind [unclassified error] cause:
			op [transport] kind [I/O error] cause:
			op [connect] kind [I/O error] cause [network unreachable]]

		e3
		[op [send email] kind [unclassified error] cause:
			op [transport] kind [I/O error] cause:
			op [connect] kind [I/O error] cause [network unreachable]:
	*/

}

func TestError_UnmarshalJSON_failures(t *testing.T) {
	tests := []struct {
		json string
	}{
		{""},
		{`invalid json`},
		{`["an","array"]`},
		{`{"op":"op1","cause":["an","invalid array}`},
	}

	for idx, test := range tests {
		t.Run(fmt.Sprint(idx, test.json), func(t *testing.T) {
			var err errors.Error
			require.Error(t, json.Unmarshal([]byte(test.json), &err), "%s", &err)
		})
	}
}

func TestMiddlewareError(t *testing.T) {
	eList := []error{
		createMoreNestedError(),
		createNestedError(),
		errors.Str("bad issue"),
	}
	midErrors := map[string]interface{}{
		"errors": eList}
	b, err := json.MarshalIndent(midErrors, "", "  ")
	require.NoError(t, err)

	fmt.Println(string(b))

	list, err := errors.UnmarshalJsonErrorList(b)
	require.NoError(t, err)
	require.Equal(t, 2, len(list.Errors))

	for _, ee := range list.Errors {
		require.True(t, strings.Contains(ee.Error(), "op"))
	}

	errStr := list.Error()
	require.Equal(t, 5, strings.Count(errStr, "op"))

	fmt.Println(errStr)
	/*
		error-list count [2]
			0: op [send email] kind [I/O error] cause:
			op [transport] kind [I/O error] cause:
			op [connect] kind [I/O error] cause [network unreachable]
			1: op [send email] kind [I/O error] cause:
			op [connect] kind [I/O error] k1 [v1] cause [network unreachable]
	*/

}

func BenchmarkError_Error_notrace(b *testing.B) {
	defer resetDefaultFieldOrder()

	tests := []struct {
		name string
		err  error
	}{
		{"no extra fields", errors.NoTrace("an op", errors.K.Invalid)},
		{"3 extra fields", errors.NoTrace("an op", errors.K.Invalid, "key1", "val1", "key2", "val2", "key3", "val3")},
		{"6 extra fields", errors.NoTrace("an op", errors.K.Invalid, "key1", "val1", "key2", "val2", "key3", "val3", "key4", "val4", "key5", "val5", "key6", "val6")},
	}

	fieldOrders := [][]string{
		nil,
		{"op", "kind", "", "cause"},
		{"op", "kind", "key5", "key1", "key6", "key4", "key2", "key3", "key7", "cause"},
	}
	for i, fieldOrder := range fieldOrders {
		fmt.Println("\n# field-order:", fieldOrder)
		errors.DefaultFieldOrder = fieldOrder

		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for _, test := range tests {
				// fmt.Println("\t", test.err)
				b.Run(test.name, func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						test.err.Error()
					}
				})
			}
		})
	}
}

type customContext struct{}

func (c *customContext) Deadline() (deadline time.Time, ok bool) {
	return
}

func (c *customContext) Done() <-chan struct{} {
	return nil
}

func (c *customContext) Err() error {
	return io.EOF
}

func (c *customContext) Value(interface{}) interface{} {
	return nil
}

func createNestedError() *errors.Error {
	err := errors.E("connect", errors.K.IO, errors.Str("network unreachable"), "k1", "v1")
	return errors.E("send email", err)
}

func createMoreNestedError() *errors.Error {
	err := errors.E().WithOp("connect").WithKind(errors.K.IO).WithCause(fmt.Errorf("network unreachable"))
	err = errors.E().WithOp("transport").WithKind(errors.K.IO).WithCause(err)
	return errors.E().WithOp("send email").WithKind(errors.K.Other).WithCause(err)
}

func enableStacktraces() func() {
	ps := errors.PrintStacktrace
	psp := errors.PrintStacktracePretty
	errors.PrintStacktrace = true
	errors.PrintStacktracePretty = true
	return func() {
		errors.PrintStacktrace = ps
		errors.PrintStacktracePretty = psp
	}
}

func enableMarshalStacktraceAsArray() func() {
	revert := enableStacktraces()
	msa := errors.MarshalStacktraceAsArray
	errors.MarshalStacktraceAsArray = true
	return func() {
		revert()
		errors.MarshalStacktraceAsArray = msa
	}
}

func resetDefaultFieldOrder() func() {
	fieldOrder := errors.DefaultFieldOrder
	return func() {
		errors.DefaultFieldOrder = fieldOrder
	}
}

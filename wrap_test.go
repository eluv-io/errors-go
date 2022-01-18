//go:build go1.13
// +build go1.13

package errors_test

import (
	"fmt"
	"io"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eluv-io/errors-go"
)

func TestUnwrap(t *testing.T) {
	var err error
	require.Nil(t, errors.Unwrap(err))

	err = createNestedError()
	fmt.Println(err)
	require.Contains(t, err.Error(), "send email")

	err = errors.Unwrap(err)
	require.Error(t, err)
	require.Contains(t, err.Error(), "connect")

	err = errors.Unwrap(err)
	require.Error(t, err)
	require.Equal(t, err.Error(), "network unreachable")

	require.Nil(t, errors.Unwrap(errors.E("noop")))

	var e *errors.Error
	require.Nil(t, e.Unwrap())
}

func TestUnwrapAll(t *testing.T) {
	// require.Nil checks for nil interfaces, hence this check succeeds
	require.Nil(t, errors.UnwrapAll(nil))
	// but in fact the return value is not nil
	require.False(t, errors.UnwrapAll(nil) == nil)
	// because it's actually equal to NilError
	require.Equal(t, errors.NilError, errors.UnwrapAll(nil))
	// which produces an empty string as Error() message
	require.Equal(t, "", errors.UnwrapAll(nil).Error())

	var err error
	err = createNestedError()
	fmt.Println(err)
	require.Contains(t, err.Error(), "send email")
	err = errors.UnwrapAll(err)
	require.Error(t, err)
	require.Equal(t, err.Error(), "network unreachable")
}

func TestAs(t *testing.T) {
	var err error
	err = &fs.PathError{}

	var pathError *fs.PathError
	require.True(t, errors.As(err, &pathError))
}

func TestIs(t *testing.T) {
	require.True(t, errors.Is(errors.E(io.EOF), io.EOF))
	require.False(t, errors.Is(errors.E(io.EOF), io.ErrUnexpectedEOF))
}

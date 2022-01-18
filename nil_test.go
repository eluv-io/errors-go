package errors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNilError(t *testing.T) {
	require.Equal(t, "", NilError.Error())
	_, ok := nilErr().(error)
	require.True(t, ok)
}

func nilErr() error {
	return NilError
}

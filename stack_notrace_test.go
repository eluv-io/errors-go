package errors_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/eluv-io/errors-go"
)

func TestPartialNestedNoStack(t *testing.T) {
	revert := enableStacktraces()
	defer revert()

	tests := []struct {
		err       error
		wantTrace bool
		match     string
	}{
		// regular: all errors have a trace
		{f1(true), true, "withTrace()"},
		// no trace
		{f1(false), true, "withTrace()"},
		// even though the top-level error has no trace, we expect the nested
		// error's trace to be printed
		{noTrace(func() error { return withTrace(nil) }), true, "withTrace()"},
		{errors.NoTrace("no trace"), false, "TestPartialNestedNoStack"},
		{noTrace(nil), false, "TestPartialNestedNoStack"},
	}

	for idx, test := range tests {
		fmt.Println(test.err)
		if test.wantTrace {
			assert.Contains(t, test.err.Error(), test.match, "#%d", idx)
		} else {
			assert.NotContains(t, test.err.Error(), test.match)
		}
	}
}

func f1(trace bool) error {
	nested := noTrace(func() error {
		return withTrace(nil)
	})

	if trace {
		return errors.E("f1", nested)
	}
	return errors.NoTrace("f1", nested)
}

func noTrace(fn func() error) error {
	if fn != nil {
		return errors.NoTrace("f2 no trace", fn())
	}
	return errors.NoTrace("f2 no trace")
}

func withTrace(fn func() error) error {
	if fn != nil {
		return errors.E("f3 with trace", fn())
	}
	return errors.E("f3 with trace")
}

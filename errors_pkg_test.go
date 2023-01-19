package errors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStacktraceToArray(t *testing.T) {
	tests := []struct {
		stacktrace string
		want       []string
	}{
		{
			stacktrace: "",
			want:       []string{},
		},
		{
			stacktrace: "\t  \t  ",
			want:       []string{},
		},
		{
			stacktrace: "abc",
			want:       []string{"abc"},
		},
		{
			stacktrace: "\n",
			want:       []string{},
		},
		{
			stacktrace: "\t\n\t \t",
			want:       []string{},
		},
		{
			stacktrace: "a\nb\nc",
			want:       []string{"a", "b", "c"},
		},
		{
			stacktrace: "\ta\n   b   \t \n\tc\n\n\t\n",
			want:       []string{"a", "b", "c"},
		},
		{
			stacktrace: "\tgithub.com/eluv-io/errors-go/errors_pkg_test.go:123 someError()\n" +
				"\tgithub.com/eluv-io/errors-go/errors_test.go:921     createNestedError()\n" +
				"\tgithub.com/eluv-io/errors-go/errors_test.go:922     createNestedError()\n" +
				"\tgithub.com/eluv-io/errors-go/errors_test.go:604     TestError_MarshalJSON()\n",
			want: []string{
				"github.com/eluv-io/errors-go/errors_pkg_test.go:123 someError()",
				"github.com/eluv-io/errors-go/errors_test.go:921     createNestedError()",
				"github.com/eluv-io/errors-go/errors_test.go:922     createNestedError()",
				"github.com/eluv-io/errors-go/errors_test.go:604     TestError_MarshalJSON()",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.stacktrace, func(t *testing.T) {
			res := stacktraceToArray(test.stacktrace)
			require.Equal(t, test.want, res)
		})
	}
}

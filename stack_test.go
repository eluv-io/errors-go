package errors_test

import (
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/eluv-io/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eluv-io/errors-go"
)

var errorLines = strings.Split(`
\tgithub.com/eluv-io/errors-go/stack_with_long_filename_test.go:\d+\s+createErrorWithExtraLongFilename\(\)
\tgithub.com/eluv-io/errors-go/stack_test.go:\d+\s+func4\(\)
\tgithub.com/eluv-io/errors-go/stack_test.go:\d+\s+func4\(\)
\tgithub.com/eluv-io/errors-go/stack_test.go:\d+\s+T.func3\(\)
\tgithub.com/eluv-io/errors-go/stack_test.go:\d+\s+T.func2\(\)
\tgithub.com/eluv-io/errors-go/stack_test.go:\d+\s+func1\(\)
\tgithub.com/eluv-io/errors-go/stack_test.go:\d+\s+TestStack.func1\(\)
`[1:], "\n")

var errorLineREs = make([]*regexp.Regexp, len(errorLines))

func init() {
	for i, s := range errorLines {
		errorLineREs[i] = regexp.MustCompile(fmt.Sprintf("^%s$", s))
	}
}

// Test that the error stack includes all the function calls between where it was generated and where it was printed. It
// should not include the name of the function in which the Error method is called. It should coalesce the call stacks
// of nested errors into one single stack, and present that stack before the other error values.
func TestStack(t *testing.T) {
	revert := enableStacktraces()
	defer revert()

	for _, psp := range []bool{true, false} {
		errors.PrintStacktracePretty = psp
		t.Run(fmt.Sprint("pretty", psp), func(t *testing.T) {
			validateStacktrace(t, func1(false).Error())
			validateStacktrace(t, func1(true).Error())
		})
	}
}

func TestClearStacktrace(t *testing.T) {
	revert := enableStacktraces()
	defer revert()

	var err error
	err = errors.ClearStacktrace(nil)
	assert.Nil(t, err)

	err = errors.ClearStacktrace(io.EOF)
	assert.Equal(t, io.EOF, err)

	err = errors.ClearStacktrace(createNestedError())
	fmt.Println(err)
	s := err.Error()
	require.NotContains(t, s, "TestClearStacktrace")
}

func validateStacktrace(t *testing.T, got string) {
	fmt.Println(got)
	lines := strings.Split(got, "\n")
	lines = lines[4:] // remove error line ("op [] ...") from error and nested error - see func1()
	for i, re := range errorLineREs {
		if i >= len(lines) {
			// Handled by line number check.
			break
		}
		if !re.MatchString(lines[i]) {
			t.Errorf("error does not match at line %v, got:\n\t%q\nwant:\n\t%s", i, lines[i], re)
		}
	}
	// Check number of lines after checking the lines themselves,
	// as the content check will likely be more illuminating.
	if got, want := len(lines), len(errorLines); got != want {
		t.Errorf("got %v lines of errors, want %v", got, want)
	}
}

/*
  func1 causes an error of the form:

	op [some operation] Kind [unclassified error] cause:
		op [GetKey] Kind [unclassified error] cause:
		op [long-error] Kind [unclassified error] cause:
		op [origin] Kind [unclassified error]
		github.com/eluv-io/errors-go/stack_with_long_filename_test.go:6 createErrorWithExtraLongFilename()
		github.com/eluv-io/errors-go/stack_test.go:120                  func4()
		github.com/eluv-io/errors-go/stack_test.go:121                  func4()
		github.com/eluv-io/errors-go/stack_test.go:116                  T.func3()
		github.com/eluv-io/errors-go/stack_test.go:111                  T.func2()
		github.com/eluv-io/errors-go/stack_test.go:102                  func1()
		github.com/eluv-io/errors-go/stack_test.go:46                   TestStack.func1()
*/
func func1(useCauseFn bool) error {
	var t T
	return t.func2(useCauseFn)
}

type T struct{}

func (t T) func2(useCauseFn bool) error {
	if useCauseFn {
		return errors.E().WithOp("some operation").WithCause(t.func3())
	} else {
		return errors.E("some operation", t.func3())
	}
}

func (T) func3() error {
	return func4()
}

func func4() error {
	err := createErrorWithExtraLongFilename(errors.NoTrace("origin"))
	return errors.E().WithOp("GetKey").WithCause(err)
}

/*
	$ go test -v -bench . -benchtime 10s -run "^Benchmark" github.com/eluv-io/errors-go
	goos: darwin
	goarch: amd64
	pkg: github.com/eluv-io/errors-go
	BenchmarkPrintNoStack-8        	 4475152	      2684 ns/op
	BenchmarkPrintStackPretty-8    	 1278406	      9282 ns/op
	BenchmarkPrintStackRegular-8   	 1668196	      7133 ns/op
	PASS
	ok  	github.com/eluv-io/errors-go	55.260s
*/

func BenchmarkPrintNoStack(b *testing.B) {
	err := errors.ClearStacktrace(createMoreNestedError())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkPrintStackPretty(b *testing.B) {
	doBenchmarkPrintStack(b, true)
}

func BenchmarkPrintStackRegular(b *testing.B) {
	doBenchmarkPrintStack(b, false)
}

func doBenchmarkPrintStack(b *testing.B, pretty bool) {
	revert := enableStacktraces()
	defer revert()

	prev := errors.PrintStacktracePretty
	defer func() {
		errors.PrintStacktracePretty = prev
	}()

	errors.PrintStacktracePretty = pretty

	err := createMoreNestedError()
	//fmt.Println(err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}

}

func BenchmarkPopulateStack(b *testing.B) {
	b.Run("stack.Trace", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s := stack.Trace()
			if false {
				fmt.Println(s)
			}
		}
	})
	b.Run("runtime.Callers", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var pcs [512]uintptr
			n := runtime.Callers(1, pcs[:])
			if false {
				fmt.Println(n)
			}
		}
	})
	b.Run("callers", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			pcs := callers()
			if false {
				fmt.Println(pcs)
			}
		}
	})
}

func callers() []uintptr {
	var pcs [512]uintptr
	n := runtime.Callers(1, pcs[:])
	return pcs[:n]
}

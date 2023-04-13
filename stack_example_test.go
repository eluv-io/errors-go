package errors_test

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/eluv-io/errors-go"
)

func readFile(filename string) error {
	_, err := os.ReadFile(filename)
	if err != nil {
		return errors.E("readFile", errors.K.IO, err, "filename", filename)
	}
	return nil
}

func getConfig() error {
	err := readFile("illegal-filename|*")
	if err != nil {
		return errors.E("getConfig", errors.K.Invalid.Default(), err)
	}
	return nil
}

func ExampleE() {
	reset := enableStacktraces()
	defer reset()

	err := getConfig()
	printError(err)

	// Output:
	//
	// op [getConfig] kind [I/O error] cause:
	//	op [readFile] kind [I/O error] filename [illegal-filename|*] cause [open illegal-filename|*: no such file or directory]
	//	github.com/eluv-io/errors-go/stack_example_test.go:   readFile()
	//	github.com/eluv-io/errors-go/stack_example_test.go:   getConfig()
	//	github.com/eluv-io/errors-go/stack_example_test.go:   getConfig()
	//	github.com/eluv-io/errors-go/stack_example_test.go:   ExampleE()
}

func ExampleTemplateFn_IfNotNil() {
	reset := enableStacktraces()
	defer reset()

	e := errors.Template("example", errors.K.Invalid, "key", "value")
	printError(e.IfNotNil(io.EOF))

	// Output:
	//
	// op [example] kind [invalid] key [value] cause [EOF]
	//	github.com/eluv-io/errors-go/stack_example_test.go:   ExampleTemplateFn_IfNotNil()
}

func ExampleTemplateFn_Add() {
	reset := enableStacktraces()
	defer reset()

	e := errors.Template("example", errors.K.Invalid)
	e = e.Add("key", "value")
	printError(e(io.EOF))

	// Output:
	//
	// op [example] kind [invalid] key [value] cause [EOF]
	//	github.com/eluv-io/errors-go/stack_example_test.go:   ExampleTemplateFn_Add()
}

func printError(err error) {
	fmt.Println(replaceLineNumbersWithBlank(deleteLastLines(err.Error(), " Example")))
}

func deleteLastLines(s string, match string) string {
	s = strings.TrimRight(s, "\n")
	for {
		pos := strings.LastIndexByte(s, '\n')
		if pos >= 0 {
			if strings.Contains(s[pos:], match) {
				break
			}
			s = s[:pos]
		} else {
			break
		}
	}
	return s
}

func replaceLineNumbersWithBlank(s string) string {
	re := regexp.MustCompile(`\.go:(\d+) `)
	bts := []byte(s)
	idxs := re.FindAllSubmatchIndex(bts, -1)
	for _, idx := range idxs {
		if len(idx) >= 4 {
			for i := idx[2]; i < idx[3]; i++ {
				bts[i] = ' '
			}
		}
	}
	return string(bts)
}

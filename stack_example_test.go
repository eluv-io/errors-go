package errors_test

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/eluv-io/errors-go"
)

func readFile(filename string) error {
	_, err := ioutil.ReadFile(filename)
	if err != nil {
		return errors.E("readFile", errors.K.IO, err, "filename", filename)
	}
	return nil
}

func getConfig() error {
	err := readFile("illegal-filename|*")
	if err != nil {
		return errors.E("getConfig", errors.K.Invalid, err)
	}
	return nil
}

func ExampleE() {
	reset := enableStacktraces()
	defer reset()

	err := getConfig()
	fmt.Println(deleteLastLines(err.Error(), "ExampleE()"))

	// Output:
	//
	// op [getConfig] kind [invalid] cause:
	//	op [readFile] kind [I/O error] filename [illegal-filename|*] cause [open illegal-filename|*: no such file or directory]
	//	github.com/eluv-io/errors-go/stack_example_test.go:14 readFile()
	//	github.com/eluv-io/errors-go/stack_example_test.go:20 getConfig()
	//	github.com/eluv-io/errors-go/stack_example_test.go:22 getConfig()
	//	github.com/eluv-io/errors-go/stack_example_test.go:31 ExampleE()
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

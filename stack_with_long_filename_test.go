package errors_test

import "github.com/eluv-io/errors-go"

func createErrorWithExtraLongFilename(cause error) error {
	return errors.E("long-error", cause)
}

package errors

// NilError is an error that represents a "nil" error value, but allows to call the Error()
// method without panicking. NilError.Error() returns the empty string "".
var NilError = (*nilError)(nil)

type nilError struct{}

func (n *nilError) Error() string {
	return ""
}

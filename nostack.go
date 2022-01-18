//go:build errnostack

package errors

import "bytes"

// stack is a noop implementation that disables stack collection & printing when the errnostack build tag is set. See
// stack.go for futher information.
type stack struct{}

func (e *Error) populateStack()               {}
func (e *Error) printStack(*bytes.Buffer)     {}
func (e *Error) dropStackFrames(n int) *Error { return e }
func (e *Error) hasStack() bool               { return false }
func (e *Error) clearStack()                  {}

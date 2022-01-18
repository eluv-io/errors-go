//go:build !errnostack

package errors

import (
	"bytes"
	"fmt"

	gostack "github.com/eluv-io/stack"
)

// stack is a type that is embedded in an Error struct, and contains information about the call stack that created that
// Error.
type stack struct {
	pcs   []uintptr         // the program counters returned by runtime.Callers()
	trace gostack.CallStack // the call stack - only filled in when needed.
}

// populateStack uses the runtime to populate the Error's stack struct with information about the current stack. It
// should be called from the E function, when the Error is being created.
//
// If the Error has another Error value in its Cause field, populateStack coalesces the stack from the inner error (if
// any) with the current stack, so that any given Error value only prints one stack.
func (e *Error) populateStack() {
	// 2 removes the populateStack() and E() or Cause() functions
	e.pcs = gostack.Callers(2)
}

// dropStackFrames removes the top n stack frames.
func (e *Error) dropStackFrames(n int) *Error {
	if len(e.pcs) > n {
		e.pcs = e.pcs[n:]
		e.trace = nil
	}
	return e
}

// printStack formats and prints the stack for this Error to the given buffer. It should be called from the Error's
// Error method.
func (e *Error) printStack(b *bytes.Buffer) {
	trace := e.coalesceStack()
	if PrintStacktracePretty {
		filenames := make([]string, len(trace))
		max := 0
		for i, call := range trace {
			filenames[i] = fmt.Sprintf("%+v", call)
			fl := len(filenames[i])
			if max < fl {
				max = fl
			}
		}
		for i, call := range trace {
			fmt.Fprintf(b, "\t%-*s %n()\n", max, filenames[i], call)
		}
		return
	}
	for _, call := range trace {
		fmt.Fprintf(b, "\t%+v\t%[1]n()\n", call)
	}
}

func (e *Error) coalesceStack() gostack.CallStack {
	if e.trace == nil && e.pcs != nil {
		e.trace = gostack.TraceFrom(e.pcs).TrimRuntime()
	}

	e2, ok := e.cause.(*Error)
	if !ok {
		return e.trace
	}

	ct := combineCallStacks(e.trace, e2.coalesceStack())
	return ct
}

// hasStack returns true if this error or any nested error has a stack trace, false otherwise.
func (e *Error) hasStack() bool {
	if e.pcs != nil {
		return true
	}
	e2, ok := e.cause.(*Error)
	if ok {
		return e2.hasStack()
	}
	return false
}

func (e *Error) clearStack() {
	e.pcs = nil
	e.trace = nil
}

func combineCallStacks(c1, c2 gostack.CallStack) gostack.CallStack {
	if c1 == nil {
		return c2
	}
	if c2 == nil {
		return c1
	}

	i := 0
	l1 := len(c1)
	l2 := len(c2)
	for ; i < l1 && i < l2; i++ {
		if !equivalent(c1[l1-1-i], c2[l2-1-i]) {
			break
		}
	}
	res := make(gostack.CallStack, l2-i+l1)
	copy(res, c2[:l2-i])
	copy(res[l2-i:], c1)
	return res
}

func equivalent(c1, c2 gostack.Call) bool {
	f1 := c1.Frame()
	f2 := c2.Frame()
	// don't compare the pc - for a line like
	//   return errors.E("op", callThatReturnsError())
	// all values are equal except for the PC
	return f1.Entry == f2.Entry &&
		f1.File == f2.File &&
		f1.Func == f2.Func &&
		f1.Function == f2.Function &&
		f1.Line == f2.Line
}

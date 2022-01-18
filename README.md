# Error Handling with `eluv-io/errors-go`

The package `eluv-io/errors-go` makes Go error handling simple, intuitive and effective.

```go
err := someFunctionThatCanFail()
if err != nil {
    return errors.E("file download", errors.K.IO, err, "file", f, "user", usr)
}
```

The `eluv-io/errors-go` package promotes and facilitates these main design principles:

* augment errors with relevant **context information** all along the call stack
* provide this additional information in a **structured form**
* provide **call stack traces** with minimal overhead if enabled through build tag and runtime config
* **integrate seamlessly** with the structured logging convention of `eluv-io/log`

For sample code, see

* [Examples](example_test.go)
* [Unit tests](errors_test.go)
* [Log sample](http://github.com/eluv-io/log-go/sample/log_sample.go)

> Note that some of the unit tests and examples use the "old-style" explicit functions `WithOp(op)`, `WithKind(kind)` and `WithCause(err)` functions to set values. They are not really needed anymore, and code can be simplified by using `errors.E()` exclusively. Furthermore, the use of templates makes the code even more compact in many situations - see sections below...

## Creating & Wrapping Errors with `E()` and `NoTrace()`

The `eluv-io/errors-go` package provides one main function to create new or wrap existing errors with additional information:

```go
// E creates a new error initialized with the given (optional) operation, kind, cause and key-value fields.
// All arguments are optional. If the first argument is a `string`, it is considered to be the operation.
//
// If no kind is specified, K.Other is assumed.
func E(args ...interface{}) *Error {...}
```

It's important to note that the first argument in E() - if it of type `string` - is the _operation_ that returns the error, and not the error's _reason_. See the following examples:

```go
// operation "file download" (op) failed with an "IO" error (kind) due to
// "err" (cause)
errors.E("file download", errors.K.IO, err)

// better than above with additional context information provided as key-value pairs
errors.E("file download", errors.K.IO, err, "file", f, "user", usr)

// sometimes there is no originating error (cause): use "reason" key-value
// pair to explain the cause
errors.E("validate configuration", errors.K.Invalid, 
	"reason", "part store location not specified")

// use "reason" for further clarification even if there is a cause,
// especially if the cause is a regular error (not an *errors.Error)
timeout, err := strconv.Atoi(timeoutString)
if err != nil {
	return errors.E("validate configuration", errors.K.Invalid, err,
		"reason", "failed to parse timeout",
		"timeout", timeoutString)
}
```

Note that supplementary context information (file & user in the example above) is provided as explicit key-value pairs instead of an opaque, custom-assembled string.

Use `errors.NoTrace()` instead of `errors.E()` to prevent recording and printing the stacktrace - see below for more information.

Besides the flexible E() function, the library's Error object also offers explicit methods for setting the different types of data:

```go
errors.E().Op("file download").Kind(errors.K.IO).Cause(err).With("file", f).With("user", usr)
```

Since that code is just more verbose, the more compact E() function call with parameters should be preferred.

## Reducing Code Complexity & Duplication with `Template()` and `TemplateNoTrace`

`Template()` or its shorter alias `T()` offers a great way to reduce code duplication in producing consistent errors. Imagine a validation function that checks multiple conditions and returns a corresponding error on any violations:

```go
if len(path) == 0 {
    return errors.E("validation", errors.K.Invalid, "id", id, "reason", "no path")
}
if strings.ContainsAny(path, `$~\:`) {
    return errors.E("validation", errors.K.Invalid, "id", id, "reason", "contains illegal characters")
}
target, err := os.Readlink(path)
if err != nil {
    return errors.E("validation", errors.K.IO, err, "id", id, "reason", "not a link")
}
```

With a template function, this code can be rewritten in a more compact and concise form:

```go
e := errors.Template("validation", errors.K.Invalid, "id", id)
if len(path) == 0 {
    return e("reason", "no path")
}
if strings.ContainsAny(path, `$~\:`) {
    return e("reason", "contains illegal characters")
}
target, err := os.Readlink(path)
if err != nil {
    return e(errors.K.IO, err, "reason", "not a link")
}
```

Use `TemplateNoTrace()` or its alias `TNoTrace` instead of `Template()` to prevent recording and printing the stacktrace in the generated error.

Use the template's `IfNotNil()` function to simplify error returns:

```go
e := errors.Template(...)
err := someFunc(...)
return e.IfNotNil(err)
```

`IfNotNil` returns nil if err is nil and instantiates and *Error according to the template otherwise. It allows also to pass additional key-value pairs, e.g. `e.IfNotNil(err, "key", "val")`

## Recording & Printing Call Stack Traces

Creating an error with `E()` or `Template()` per default also records the program counters of the current call stack, and `Error()` formats and prints the call stack:

```go
op [getConfig] kind [invalid] cause:
	op [readFile] kind [I/O error] filename [illegal-filename|*] cause [open illegal-filename|*: no such file or directory]
	github.com/eluv-io/errors-go/stack_example_test.go:13 readFile()
	github.com/eluv-io/errors-go/stack_example_test.go:19 getConfig()
	github.com/eluv-io/errors-go/stack_example_test.go:21 getConfig()
	github.com/eluv-io/errors-go/stack_example_test.go:30 ExampleE()
	testing/run_example.go:64                             runExample()
	testing/example.go:44                                 runExamples()
	testing/testing.go:1505                               (*M).Run()
	_testmain.go:165                                      main()
```

Stack traces from multiple nested `Error`s are automatically coalesced into a single, comprehensive stack trace that is printed at the end.

The following global variables control stack trace handling at runtime: 

* `PopulateStacktrace` controls whether program counters are recorded when an `Error` is created
* `PrintStacktrace` controls whether stack traces are printed in `Error.Error()`
* `PrintStacktracePretty` controls the formatting of stack traces
* `MarshalStacktrace` controls whether stack traces are marshalled to JSON

The following functions create an `Error` without stack trace:
* `errors.NoTrace()`
* `errors.TemplateNoTrace()`
* `errors.TNoTrace()`

The following functions allow to suppress or remove stack traces from an `Error`:
* `errors.ClearStacktrace()`
* `Error.FormatTrace()`
* `Error.ClearStacktrace()`

In addition, stack trace recording can be completely disabled at compile time with the `errnostack` build tag. 

## Integration with `eluv-io/log-go`

Thanks to providing all data as key-value pairs (including operation, kind and cause), the Error objects integrate seamlessly with the logging package whose main principle is that of _`structured`_ logging. This means, for example, that errors are marshaled as JSON objects when the JSON handler is used for logging:

```text
{
  "fields": {
    "error": {
      "cause": "EOF",
      "file": "/tmp/app-config.yaml",
      "kind": "I/O error",
      "op": "failed to parse config",
      "stacktrace": "runtime/asm_amd64.s:2337: runtime.goexit:\n\truntime/proc.go:195: ...main:\n\tsample/log_sample.go:66: main.main:\n\teluvio/log/sample/sub/sub.go:17: eluvio/log/sample/sub.Call"
    },
    "logger": "/eluvio/log/sample/sub"
  },
  "level": "warn",
  "timestamp": "2018-03-02T16:52:04.831737+01:00",
  "message": "failed to read config, using defaults"
}
```

Notice the "error" object: it is a JSON object with all data provided as fields. In addition, the error also contains a stacktrace when logged in JSON format.


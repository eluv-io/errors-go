package errors

// Kind is the Go type for error kinds. Use the pre-defined kinds in errors.K, or
type Kind string

// K defines the kinds of errors.
var K = struct {
	Other          Kind // Unclassified error.
	NotImplemented Kind // The functionality is not yet implemented.
	Invalid        Kind // Invalid operation for this type of item.
	Permission     Kind // Permission denied.
	IO             Kind // External I/O error such as network failure.
	Exist          Kind // Item already exists.
	NotExist       Kind // Item does not exist. Also see NotFound!
	NotFound       Kind // Item should exist but cannot be found. Also see NotExist!
	NotDir         Kind // Item is not a directory.
	Finalized      Kind // Part or content is already finalized.
	NotFinalized   Kind // Part or content is not yet finalized.
	NoNetRoute     Kind // No route found for the requested operation
	Internal       Kind // Generic internal error
	AVProcessing   Kind // error in audio/video processing
	AVInput        Kind // error encountered in media input (stall, corruption, unexpected)
	NoMediaMatch   Kind // None of the accepted media type matches the content
	Unavailable    Kind // The server cannot handle the request temporarily (e.g. overloaded or down for maintenance).
	Cancelled      Kind // The operation was cancelled.
	Timeout        Kind // The operation was timed out.
	Warn           Kind // The error is not an actual error, but rather a warning that something might be wrong.
}{
	Other:          "unclassified error",
	NotImplemented: "not implemented",
	Invalid:        "invalid",
	Permission:     "permission denied",
	IO:             "I/O error",
	Exist:          "item already exists",
	NotExist:       "item does not exist",
	NotFound:       "item cannot be found",
	Finalized:      "item is already finalized",
	NotFinalized:   "item is not finalized",
	NoNetRoute:     "no route found",
	Internal:       "internal error",
	AVProcessing:   "a/v processing error",
	AVInput:        "a/v input error",
	NoMediaMatch:   "no media type match",
	Unavailable:    "service unavailable",
	Cancelled:      "operation cancelled",
	Timeout:        "operation timed out",
	Warn:           "warning",
}

// Default turns this Kind into a default value. See DefaultKind for more information.
func (k Kind) Default() DefaultKind {
	return DefaultKind(k)
}

// DefaultKind is a Kind that can be set on an Error or Template that is only used if the kind is not otherwise set e.g.
// with an explicit call to Error.Kind(kind) or by inheriting it from a nested error.
//
//	e := errors.Template("read user", errors.K.Invalid.Default())
//  return e(nested)
//
// In the above example, the returned error will have the nested error's kind if it's defined or K.Invalid otherwise. If
// the template definition didn't use Default(), the returned error would always be K.Invalid, regardless of the kind
// in the nested error.
type DefaultKind string

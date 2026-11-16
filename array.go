package errors

import (
	"encoding/json"
)

// ErrorArray is an ErrorList that marshals to JSON as a plain array of its errors instead of an object holding the
// array in its "errors" field. MarshalJSON has a value receiver, so the array form also applies where the ErrorArray
// is not addressable, for example in a struct passed by value in an interface.
//
// Append converts the error lists nested in the appended errors to ErrorArrays, see ForJSONArrays, so that they marshal
// as arrays too. Errors set directly in the Errors field are marshalled as they are.
//
// The functions of this package that look into error lists, such as Append, IsKind and GetRootCause, recognize only an
// *ErrorList. Pass them &arr.ErrorList.
type ErrorArray struct {
	ErrorList
}

// Append appends errs like ErrorList.Append does: the errors of a list are appended instead of the list. Error lists
// nested in errs are replaced by ErrorArrays, see ForJSONArrays. errs themselves are not modified.
func (e *ErrorArray) Append(errs ...error) {
	for _, err := range errs {
		switch t := err.(type) {
		case *ErrorList:
			if t != nil {
				e.Append(t.Errors...)
			}
		case *ErrorArray:
			if t != nil {
				e.Append(t.Errors...)
			}
		default:
			if err != nil {
				converted, _ := withArrays(err)
				e.doAppend(converted)
			}
		}
	}
}

// MarshalJSON marshals the errors as an array, each in the form ForJSON returns. An empty ErrorArray marshals as null,
// like a nil slice.
func (e ErrorArray) MarshalJSON() ([]byte, error) {
	if len(e.Errors) == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(e.errorsForJSON())
}

// UnmarshalJSON unmarshals an array of errors and appends them. JSON objects become *Error, other values an error of
// their string form.
func (e *ErrorArray) UnmarshalJSON(bts []byte) error {
	var list []valOrMap
	if err := json.Unmarshal(bts, &list); err != nil {
		return err
	}
	for _, elem := range list {
		if err := elem.AsError(); err != nil {
			e.Append(err)
		}
	}
	return nil
}

// withArrays returns err with every error list in it replaced by an *ErrorArray, see ForJSONArrays. It returns err
// itself and false if err holds no list, otherwise a clone and true.
func withArrays(err error) (error, bool) {
	switch e := err.(type) {
	case *Error:
		if e == nil {
			return err, false
		}
		cause, changed := withArrays(e.cause)
		if !changed {
			return err, false
		}
		clone := *e
		clone.cause = cause
		return &clone, true
	case *ErrorList:
		if e == nil {
			return err, false
		}
		errs, _ := withArraysAll(e.Errors)
		return &ErrorArray{ErrorList: ErrorList{Errors: errs}}, true
	case *ErrorArray:
		if e == nil {
			return err, false
		}
		errs, changed := withArraysAll(e.Errors)
		if !changed {
			return err, false
		}
		return &ErrorArray{ErrorList: ErrorList{Errors: errs}}, true
	}
	return err, false
}

// withArraysAll applies withArrays to each of errs. It returns errs itself and false if none of them holds a list,
// otherwise a new slice and true.
func withArraysAll(errs []error) ([]error, bool) {
	var res []error
	for idx, err := range errs {
		converted, changed := withArrays(err)
		if changed && res == nil {
			res = make([]error, len(errs))
			copy(res, errs[:idx])
		}
		if res != nil {
			res[idx] = converted
		}
	}
	if res == nil {
		return errs, false
	}
	return res, true
}

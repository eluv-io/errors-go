package errors

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Append appends the given errs to err.
//
// If err is not a *ErrorList, then a new one is created and err added to it. Then all additional errs are appended,
// unwrapping them if any of them are ErrorLists themselves.
//
// Any nil errors within errs will be ignored. If err is nil, a new *ErrorList will be returned.
func Append(err error, errs ...error) error {
	for _, e := range errs {
		if e != nil {
			break
		}
		errs = errs[1:]
	}
	if len(errs) == 0 {
		return err
	}

	list, ok := err.(*ErrorList)
	if ok {
		//  may be a nil interface
		if list == nil {
			if len(errs) == 1 {
				return errs[0]
			}
			list = new(ErrorList)
		}
	} else {
		list = new(ErrorList)
		list.Append(err)
	}
	list.Append(errs...)
	return list.ErrorOrNil()
}

// ErrorList is a collection of errors.
type ErrorList struct {
	Errors []error
}

func (e *ErrorList) Append(errs ...error) {
	for _, err := range errs {
		switch err := err.(type) {
		case *ErrorList:
			// add nested errors instead of the list
			if err != nil {
				e.doAppend(err.Errors...)
			}
		default:
			if err != nil {
				e.doAppend(err)
			}
		}
	}
}

func (e *ErrorList) doAppend(errs ...error) {
	e.Errors = append(e.Errors, errs...)
}

// Error returns the error list as a formatted, multi-line string.
func (e *ErrorList) Error() string {
	switch len(e.Errors) {
	case 0:
		return ""
	case 1:
		return e.Errors[0].Error()
	}
	sb := strings.Builder{}
	sb.WriteString("error-list ")
	sb.WriteString("count [")
	sb.WriteString(strconv.Itoa(len(e.Errors)))
	sb.WriteString("]\n")
	for idx, err := range e.Errors {
		s := err.Error()
		sb.WriteString("\t")
		sb.WriteString(strconv.Itoa(idx))
		sb.WriteString(": ")
		sb.WriteString(s)
		sb.WriteString("\n")
	}
	return sb.String()
}

// ErrorOrNil returns an error interface if this Error represents a list of errors, or returns nil if the list of errors
// is empty.
func (e *ErrorList) ErrorOrNil() error {
	if e == nil || len(e.Errors) == 0 {
		return nil
	}
	if len(e.Errors) == 1 {
		return e.Errors[0]
	}
	return e
}

func (e *ErrorList) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{"errors": e.errorsForJSON()})
}

func (e *ErrorList) UnmarshalJSON(bts []byte) error {
	list := struct {
		Errors []valOrMap `json:"errors"`
	}{}

	err := json.Unmarshal(bts, &list)
	if err != nil {
		return err
	}

	for _, elem := range list.Errors {
		var ee error
		ee = elem.AsError()
		if ee != nil {
			e.Append(ee)
		}
	}
	return nil
}

// errorsForJSON returns the list of errors in a form apt for JSON marshalling. Specifically, it replaces errors
// implementing the standard "error" interface with their string implementation, because otherwise they would be
// marshalled to nil by json.Marshal().
func (e *ErrorList) errorsForJSON() []interface{} {
	errors := e.Errors // local copy to prevent concurrency issues
	res := make([]interface{}, len(errors))
	for idx, err := range errors {
		con, _ := convertForJSONMarshalling(err)
		res[idx] = con
	}
	return res
}

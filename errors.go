package aeio

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
)

const (
	errDatastore = "error_datastore"
	errKey       = "error_key"
	errMarshal   = "error_marshal"
	errRequest   = "error_request"
	errResource  = "error_resource"
	errResponse  = "error_response"
	errSchema    = "error_schema"
	errHttpCode  = "error_http_code"
	errUnknown   = "error_unknown"
	errUnmarshal = "error_unmarshal"
)

var (
	errorInvalidPath = &complexError{
		Name: errKey,
		Desc: "The path used is not valid",
		Code: http.StatusBadRequest,
	}
	errorDatastorePut = &complexError{
		Name: errDatastore,
		Desc: "Could not put to datastore",
		Code: http.StatusInternalServerError,
	}
	errorDatastoreRead = &complexError{
		Name: errDatastore,
		Desc: "Could not get from datastore",
		Code: http.StatusNotFound,
	}
	errorDatastoreDelete = &complexError{
		Name: errDatastore,
		Desc: "Could not delete in datastore",
		Code: http.StatusInternalServerError,
	}
	errorDatastoreCount = &complexError{
		Name: errDatastore,
		Code: http.StatusInternalServerError,
	}
	errorDatastoreAncestorNotFound = &complexError{
		Name: errDatastore,
		Desc: "The ancestor for this key was not found",
		Hint: "Verify that the path is valid",
		Code: http.StatusBadRequest,
	}
	errorDatastoreInvalidCursor = &complexError{
		Name: errDatastore,
		Desc: "The cursor passed on the request is not valid for this datastore",
		Hint: "Restart the listing pagination to get valid cursors",
		Code: http.StatusBadRequest,
	}
	errorInvalidHttpStatusCode = &complexError{
		Name: errHttpCode,
		Code: http.StatusBadRequest,
	}
	errorResponseMarshal = &complexError{
		Name: errMarshal,
		Code: http.StatusInternalServerError,
	}
	errorRequestUnmarshal = &complexError{
		Name: errUnmarshal,
		Desc: "The body could not be interpreted as json and unmarshaled to data object",
		Code: http.StatusBadRequest,
	}
	errorResponseWrite = &complexError{
		Name: errResponse,
		Code: http.StatusInternalServerError,
	}
	errorUnknown = &complexError{
		Name: errUnknown,
		Code: http.StatusInternalServerError,
		Desc: "Got some error not well described by the framework",
	}
	errorResourceModelNotImplemented = &complexError{
		Name: errResource,
		Code: http.StatusBadRequest,
		Desc: "The path has objects not implemented",
		Hint: "Verify that the path requested has all elements implemented and or registered correctly",
	}
	errorInvalidResourceChild = &complexError{
		Name: errSchema,
		Desc: "The path has invalid objects hierarchy",
		Code: http.StatusBadRequest,
		Hint: "Verify that the path requested has all elements implemented and or registered correctly",
	}
	errorEmptyRequestBody = &complexError{
		Name: errRequest,
		Desc: "There was no data found on the request body",
		Hint: "Check that the request is of the right method and valid json",
		Code: http.StatusBadRequest,
	}
	errorRequestBodyRead = &complexError{
		Name: errRequest,
		Desc: "Error reading request body",
		Hint: "Verify the presence of invalid characters",
		Code: http.StatusBadRequest,
	}
)

type complexError struct {
	Name  string `json:"name"`                  // limited error name strings, like codes for mapping
	Desc  string `json:"description"` // improved description of the problem for humans
	Hint  string `json:"hint"`        // if it is a common user mistake try to educate
	Where string `json:"stack"`
	Code  int    `json:"-"`               // http status code caused by this error
	Debug string `json:"debug"` // original error message
	cause error  // original error, only added by .withCause()
}

func (e complexError) Error() string {
	return fmt.Sprintf("%s, %s, %s, %d, %s", e.Name, e.Desc, e.Hint, e.Code, e.Debug)
}

func (e complexError) withCause(cause error) complexError {
	e.cause = cause
	e.Debug = cause.Error()
	return e
}

func (e complexError) withCode(code int) complexError {
	e.Code = code
	return e
}

func (e complexError) withStack() complexError {
	_, file, line, _ := runtime.Caller(1)
	e.Where = fmt.Sprintf("%s:%d", file, line)
	return e
}

func (e complexError) withLog() complexError {
	log.Print(e.Error())
	return e
}

func (e complexError) withHint(hint string) complexError {
	e.Hint = hint
	return e
}

// func NewError(name string, err error, statusCode int, s ...string) error {
// 	var desc, hint string
// 	if len(s) > 0 {
// 		desc = s[0]
// 	} else if len(s) > 1 {
// 		hint = s[1]
// 	}
//
// 	_, file, line, _ := runtime.Caller(1)
//
// 	e := complexError{
// 		Name:  name,
// 		Desc: desc,
// 		Hint: hint,
// 		Debug: err.Error(),
// 		Where: fmt.Sprintf("%s:%d", file, line),
// 		Code:  statusCode,
// 		cause: err,
// 	}
//
// 	log.Print(e.Error())
// 	return e
// }

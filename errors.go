package aeio

import (
	"fmt"
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
	errorWritingDatastore = &complexError{
		Name: errDatastore,
		Desc: "When putting the data object to datastore",
		Code: http.StatusInternalServerError,
	}
	errorWritingDatastoreNoData = &complexError{
		Name: errDatastore,
		Desc: "No data was found to put to datastore",
		Code: http.StatusBadRequest,
	}
	errorInvalidKey = &complexError{
		Name: errKey,
		Desc: "The path used causes an invalid key to be formed",
		Code: http.StatusBadRequest,
	}
	errorReadDatastoreEntityNotFound = &complexError{
		Name: errDatastore,
		Desc: "The element was not found in the datastore",
		Code: http.StatusNotFound,
	}
	errorDeleteDatastoreEntity = &complexError{
		Name: errDatastore,
		Code: http.StatusInternalServerError,
	}
	errorDatastoreCount = &complexError{
		Name: errDatastore,
		Code: http.StatusInternalServerError,
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
	errorBadCustomError = &complexError{
		Name: errUnknown,
		Code: http.StatusInternalServerError,
		Desc: "Got some error not well described by the framework",
		Hint: "Use aeio.NewError to extend simple errors",
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
	errorInvalidModelFunction = &complexError{
		Name: errResource,
		Desc: "Model does not implement this function",
		Hint: "Check that the function is well registered for the model",
		Code: http.StatusInternalServerError,
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
		Hint: "Verify the presence of invalid characters for utf8",
		Code: http.StatusBadRequest,
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
)

type complexError struct {
	Name  string `json:"name"`                  // limited error name strings, like codes for mapping
	Desc  string `json:"description,omitempty"` // improved description of the problem for humans
	Hint  string `json:"hint,omitempty"`        // if it is a common user mistake try to educate
	Where string `json:"-"`
	Code  int    `json:"-"`               // http status code caused by this error
	Debug string `json:"debug,omitempty"` // original error message
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
	// log.Print(e.Error())
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
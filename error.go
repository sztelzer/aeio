package aeio

import (
	"errors"
	"fmt"
	"google.golang.org/appengine/log"
	"runtime"
)

type E struct {
	Reference string `json:"reference"`
	Error     string `json:"error"`
}

func (e *E) SimpleError() error {
	return errors.New(fmt.Sprintf("%v: %v", e.Reference, e.Error))
}

func (r *Resource) E(reference string, info interface{}) {
	_, file, line, _ := runtime.Caller(1)
	err := fmt.Sprintf("%v - %v:%v", info, file, line)
	log.Errorf(*r.Access.Context, "%v: %+v", reference, err)
	r.Errors = append(r.Errors, &E{Reference: reference, Error: err})
	r.ErrorAction()
}

func (r *Resource) E1(reference string, info interface{}) {
	_, file, line, _ := runtime.Caller(1)
	err := fmt.Sprintf("%v - %v:%v", info, file, line)
	log.Errorf(*r.Access.Context, "%v: %+v", reference, err)
	r.Errors = append([]*E{{Reference: reference, Error: err}}, r.Errors...)
	r.ErrorAction()
}

func (r *Resource) L(reference string, info interface{}) {
	err := fmt.Sprintf("%+v", info)
	log.Infof(*r.Access.Context, "%v: %+v", reference, err)
}

func (r *Resource) HasErrors() bool {
	if len(r.Errors) > 0 {
		return true
	}
	return false
}

func (r *Resource) Ok() bool {
	return !r.HasErrors()
}

func (r *Resource) GetErrors(subResource *Resource) bool {
	if subResource.HasErrors() {
		r.Errors = append(r.Errors, subResource.Errors...)
		return true
	}
	return false
}

func (r *Resource) OneError() error {
	if r.HasErrors() {
		return r.Errors[0].SimpleError()
	}
	return nil
}

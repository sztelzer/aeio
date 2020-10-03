package aeio

import (
	"errors"
	"fmt"
	"log"
	"runtime"
)

type Error struct {
	Reference string `json:"reference"`
	Error     string `json:"Error"`
}

func (r *Resource) Error(reference string, info interface{}) {
	_, file, line, _ := runtime.Caller(1)
	err := fmt.Sprintf("%v - %v:%v", info, file, line)
	log.Printf("%v: %+v", reference, err)
	r.Errors = append(r.Errors, Error{Reference: reference, Error: err})
	r.ErrorAction()
}

func (r *Resource) ErrorFirst(reference string, info interface{}) {
	_, file, line, _ := runtime.Caller(1)
	err := fmt.Sprintf("%v - %v:%v", info, file, line)
	log.Printf("%v: %+v", reference, err)
	r.Errors = append([]Error{{Reference: reference, Error: err}}, r.Errors...)
	r.ErrorAction()
}

func (r *Resource) Log(reference string, info interface{}) {
	err := fmt.Sprintf("%+v", info)
	log.Printf("%v: %+v", reference, err)
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

func (r *Resource) ImportErrors(b *Resource) bool {
	if b.HasErrors() {
		r.Errors = append(r.Errors, b.Errors...)
		return true
	}
	return false
}

func (e *Error) SimpleError() error {
	return errors.New(fmt.Sprintf("%v: %v", e.Reference, e.Error))
}

func (r *Resource) OneError() error {
	if r.HasErrors() {
		return r.Errors[0].SimpleError()
	}
	return nil
}

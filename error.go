package aeio

import (
	"fmt"
	"google.golang.org/appengine/log"
)

func (r *Resource) E(f string, e error) {
	s := fmt.Sprintf("%v", e)
	n := E{
		Reference: f,
		Error:     s,
	}
	r.Errors = append(r.Errors, &n)
	r.L(f, e)
}

type E struct {
	Reference string `json:"reference"`
	Error     string `json:"error"`
}

func (r *Resource) L(f string, e error) {
	log.Infof(*r.Access.Context, f+": %v", e)
}

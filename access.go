package aeio

import (
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
)

type Access struct {
	Request  *http.Request
	Context  *context.Context
	Writer   http.ResponseWriter
	UserMeta interface{}
}

func NewAccess(writer *http.ResponseWriter, request *http.Request) (access *Access) {
	access = new(Access)
	access.Request = request
	context := appengine.NewContext(request)
	access.Context = &context
	access.Writer = *writer
	return
}

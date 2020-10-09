package aeio

import (
	"net/http"
)

type Access struct {
	Request *http.Request
	Writer  http.ResponseWriter
}

func newAccess(writer *http.ResponseWriter, request *http.Request) *Access {
	return &Access{
		Request: request,
		Writer:  *writer,
	}
}

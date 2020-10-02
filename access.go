package aeio

import (
	"cloud.google.com/go/datastore"
	"context"
	"google.golang.org/appengine"
	"net/http"
)

type Access struct {
	Request   *http.Request
	Writer    http.ResponseWriter
	Context   *context.Context
	Datastore *datastore.Client
	//UserMeta  interface{}
}

func newAccess(writer *http.ResponseWriter, request *http.Request) *Access {
	requestContext := appengine.NewContext(request)
	datastoreClient, err := datastore.NewClient(requestContext, datastore.DetectProjectID)
	if err != nil {
		panic(err)
	}

	return &Access{
		Request:   request,
		Writer:    *writer,
		Context:   &requestContext,
		Datastore: datastoreClient,
	}
}

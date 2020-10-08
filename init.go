package aeio

import (
	"cloud.google.com/go/datastore"
	"context"
	"log"
)

var ListSizeDefault = 20
var ListSizeMax = 100
var Context context.Context
var ContextCancel context.CancelFunc
var DatastoreClient *datastore.Client

func init() {
	var err error
	Context, ContextCancel = context.WithCancel(context.Background())
	DatastoreClient, err = datastore.NewClient(Context, datastore.DetectProjectID)
	if err != nil {
		log.Fatal(err)
	}
}

func ShutdownApplication() {
	err := DatastoreClient.Close()
	if err != nil {
		log.Print("error_closing_datastore_client:", err)
	}

	ContextCancel()
}
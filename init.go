package aeio

import (
	"cloud.google.com/go/datastore"
	"context"
	"log"
)

var ListSizeDefault = 20
var ListSizeMax = 100
var InstanceContext context.Context
var ShutdownContext context.CancelFunc
var DatastoreClient *datastore.Client

func init() {
	var err error
	InstanceContext, ShutdownContext = context.WithCancel(context.Background())
	DatastoreClient, err = datastore.NewClient(InstanceContext, datastore.DetectProjectID)
	if err != nil {
		log.Fatal(err)
	}
}

func Shutdown() {
	err := DatastoreClient.Close()
	if err != nil {
		log.Print("error_closing_datastore_client:", err)
	}

	ShutdownContext()
}
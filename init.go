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
		defer ShutdownContext()
		log.Fatal(err)
	}
}

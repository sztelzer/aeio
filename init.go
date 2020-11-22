package aeio

import (
	"cloud.google.com/go/datastore"
	"context"
	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
	"log"
)

var ListSizeDefault = 20
var ListSizeMax = 100
var Context context.Context
var ContextCancel context.CancelFunc
var DatastoreClient *datastore.Client

var FireApp *firebase.App
var FireAppAuthClient *firebaseAuth.Client

func init() {
	var err error

	Context, ContextCancel = context.WithCancel(context.Background())
	DatastoreClient, err = datastore.NewClient(Context, datastore.DetectProjectID)
	if err != nil {
		log.Fatalf("error initializing datastore client: %v", err)
	}

	// in localhost it gets the key.json from ENV
	FireApp, err = firebase.NewApp(Context, nil)
	if err != nil {
		log.Fatalf("error initializing firebase app: %v", err)
	}

	FireAppAuthClient, err = FireApp.Auth(Context)
	if err != nil {
		log.Fatalf("error initializing firebase auth client: %v", err)
	}

}

// ShutdownApplication should be called anytime the app exits to close services connections.
// Recommended to be put as a deferred call on the start of the main application.
// ie.:
// `func main() {
//	defer aeio.ShutdownApplication()
//  ... middleware and routes ...
// 	err = aeio.Serve(router)
//	log.Fatal(err)
// }
func ShutdownApplication() {
	err := DatastoreClient.Close()
	if err != nil {
		log.Print("error_closing_datastore_client:", err)
	}

	ContextCancel()
}
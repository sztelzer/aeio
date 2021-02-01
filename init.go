package aeio

import (
	"context"
	"log"
	"os"
	
	"cloud.google.com/go/datastore"
	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
)

var Development = false

var ListSizeDefault = 20
var ListSizeMax = 100
var Context context.Context
var ContextCancel context.CancelFunc
var DatastoreClient *datastore.Client

var FireApp *firebase.App
var FireAppAuthClient *firebaseAuth.Client

// On localhost with emulators, set these ENV before running the application
// DATASTORE_EMULATOR_HOST=localhost:8081;ENVIRONMENT=DEVELOPMENT;GOOGLE_APPLICATION_CREDENTIALS=../gaeio2-firebase-adminsdk-82w3s-69fc074ae2.json

func init() {
	var err error
	
	if os.Getenv("DEVELOPMENT") == "true" {
		Development = true
		log.Println("Initializing App as DEVELOPMENT")
	}
	
	Context, ContextCancel = context.WithCancel(context.Background())
	DatastoreClient, err = datastore.NewClient(Context, datastore.DetectProjectID)
	if err != nil {
		log.Fatalf("error initializing datastore client: %v", err)
	}
	
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

package aeio

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/datastore"
	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
)

// Development holds the ambient definition of being in development mode. It is set by the ENV variable 'DEVELOPMENT'.
// Use it to change behavior between production and development runtimes.
var Development = false

// QuerySizeDefault sets the default size of pagination on list requests. You may change it. It will be used for requests that don't sets a size.
var QuerySizeDefault = 100

// QuerySizeMax sets the maximum size of a list request. You may change it. It will be enforced as a hard limit to requests that do send a size.
var QuerySizeMax = 1000

// Context holds the server base context. Use it to generate other contexts when needed.
var Context context.Context

// ContextCancel is the function that cancels the general context. You can use it to send the cancellation of context to all children contexts.
// Useful for graceful shutdown.
// We also provide a more complete way to shutdown the application with ShutdownApplication().
var ContextCancel context.CancelFunc

// DatastoreClient is the global default singleton datastore client. Use it everytime that you need to access the datastore by yourself.
// It uses the main context, so it will respect the the context cancellation.
var DatastoreClient *datastore.Client

// FireApp it the default Firebase App. You may use it to access different services of firebase.
// We already provide a client for Authentication: the FireAppAuthClient
var FireApp *firebase.App

// FireAppAuthClient is the Firebase Auth Client.
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
//	  defer aeio.ShutdownApplication()
//    ... middleware and routes ...
// 	  err = aeio.Serve(router)
//	  log.Fatal(err)
// }
func ShutdownApplication() {
	err := DatastoreClient.Close()
	if err != nil {
		log.Print("error_closing_datastore_client:", err)
	}

	ContextCancel()
}

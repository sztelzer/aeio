package aeio

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var ServerHost = ""
var ServerPort = "8080"

func Serve(router http.Handler) error {
	var err error

	if Development {
		ServerHost = "127.0.0.1"
	}

	port := os.Getenv("PORT")
	if port != "" {
		ServerPort = port
		// ServerHost = ""
	}
	connectionString := fmt.Sprintf("%s:%s", ServerHost, ServerPort)

	log.Printf("Serving HTTP on %s", connectionString)
	err = http.ListenAndServe(connectionString, router)
	// if listenAndServe is ok, will stay running (blocking)
	// returns only in error (withCause will always be true)
	return err
}

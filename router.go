package aeio

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var ServerHost string = "localhost"
var ServerPort string = "8080"


func Serve(router http.Handler) {
	if p := os.Getenv("PORT"); p != "" {
		ServerPort = p
		ServerHost = ""
	}
	connectionString := fmt.Sprintf("%s:%s", ServerHost, ServerPort)

	log.Printf("Serving HTTP on port %s", ServerPort)
	log.Fatal(http.ListenAndServe(connectionString, router))
}


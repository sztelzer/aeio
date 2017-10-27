package aeio

import (
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {
	return mux.NewRouter()
}

type WithCORS struct {
	Router *mux.Router
}

func (s *WithCORS) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	origin := req.Header.Get("Origin")
	if origin != "" {
		res.Header().Set("Access-Control-Allow-Origin", origin)
		res.Header().Set("Access-Control-Allow-Methods", "POST, GET, PATCH, DELETE, PUT, OPTIONS")
		res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		res.Header().Set("Content-Type", "application/json")
	}

	// Stop here for a Preflighted OPTIONS request.
	if req.Method == "OPTIONS" {
		return
	}
	// Lets Gorilla work
	s.Router.ServeHTTP(res, req)
}

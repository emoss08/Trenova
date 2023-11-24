package router

import (
	"backend/handlers"
	"backend/middleware"

	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func InitRouter() *mux.Router {
	r := mux.NewRouter()

	// Apply the LoggingMiddleware globally
	r.Use(middleware.LoggingMiddleware)

	// Apply the RecoveryHandler globally
	r.Use(gh.RecoveryHandler())

	// Apply the CompressHandler globally
	r.Use(gh.CompressHandler)

	// Create a subrouter for the /v1/ path
	apiV1 := r.PathPrefix("/api/v1").Subrouter()

	// Deinfe test endpoint
	apiV1.HandleFunc("/test", handlers.TestEndpoint).Methods("POST")

	return r
}

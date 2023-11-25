package router

import (
	"backend/handlers"
	"backend/middleware"
	"backend/service"

	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func InitRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()

	// Apply the LoggingMiddleware globally
	r.Use(middleware.LoggingMiddleware)

	// Apply the RecoveryHandler globally
	r.Use(gh.RecoveryHandler())

	// Apply the CompressHandler globally
	r.Use(gh.CompressHandler)

	// Initialize ServiceContainer
	s := service.InitializeServices(db)

	// Create a subrouter for the /v1/ path
	apiV1 := r.PathPrefix("/api/v1").Subrouter()

	// Deinfe test endpoint
	apiV1.HandleFunc("/test", handlers.TestEndpoint).Methods("POST")

	// User API endpoints
	apiV1.HandleFunc("/users", handlers.CreateUserHandler(s)).Methods("POST")
	apiV1.HandleFunc("/users", handlers.GetAllUsersHandler(s)).Methods("GET")
	return r
}

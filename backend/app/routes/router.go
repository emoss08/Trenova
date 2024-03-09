package routes

import (
	"trenova-go-backend/app/middleware"

	"github.com/gorilla/mux"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func InitializeRouter(db *gorm.DB, store *gormstore.Store) *mux.Router {
	r := mux.NewRouter()

	r.Use(middleware.AdvancedLoggingMiddleware) // Logging middleware

	// Register routes
	apiRouter := r.PathPrefix("/api").Subrouter()

	// Register sub-routes
	AuthRoutes(apiRouter, db, store)         // Auth routes
	RevenueCodeRoutes(apiRouter, db, store)  // RevenueCode routes
	OrganizationRoutes(apiRouter, db, store) // Organization routes
	UserRoutes(apiRouter, db, store)         // User routes

	// Log all available routes
	// r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
	// 	t, _ := route.GetPathTemplate()
	// 	m, _ := route.GetMethods()
	// 	fmt.Println("ROUTE:", t, m)
	// 	return nil
	// })

	return r
}

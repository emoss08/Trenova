package routes

import (
	"trenova-go-backend/app/middleware"

	"github.com/gorilla/mux"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func InitializeRouter(db *gorm.DB, store *gormstore.Store) *mux.Router {
	r := mux.NewRouter()

	apiRouter := r.PathPrefix("/api").Subrouter()

	AuthRoutes(apiRouter, db, store)

	protectedRouter := apiRouter.NewRoute().Subrouter()
	protectedRouter.Use(middleware.SessionMiddleware(store))

	OrganizationRoutes(protectedRouter, db)
	RevenueCodeRoutes(protectedRouter, db) // RevenueCode routes
	UserRoutes(protectedRouter, db)        // User routes

	// Log all available routes
	// r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
	// 	t, _ := route.GetPathTemplate()
	// 	m, _ := route.GetMethods()
	// 	fmt.Println("ROUTE:", t, m)
	// 	return nil
	// })

	return r
}

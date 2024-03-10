package routes

import (
	"trenova/app/middleware"

	"github.com/gorilla/mux"
	"github.com/henvic/httpretty"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func InitializeRouter(db *gorm.DB, store *gormstore.Store) *mux.Router {
	r := mux.NewRouter()

	apiRouter := r.PathPrefix("/api").Subrouter()

	AuthRoutes(apiRouter, db, store)
	logger := &httpretty.Logger{
		Time:           true,
		TLS:            true,
		RequestHeader:  true,
		RequestBody:    true,
		ResponseHeader: true,
		ResponseBody:   true,
		Colors:         true, // erase line if you don't like colors
	}

	protectedRouter := apiRouter.NewRoute().Subrouter()
	protectedRouter.Use(middleware.SessionMiddleware(store))
	protectedRouter.Use(logger.Middleware)

	OrganizationRoutes(protectedRouter, db)      // Organization Routes
	AccountingControlRoutes(protectedRouter, db) // AccountingControl routes
	RevenueCodeRoutes(protectedRouter, db)       // RevenueCode routes
	UserRoutes(protectedRouter, db)              // User routes

	// Log all available routes
	// r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
	// 	t, _ := route.GetPathTemplate()
	// 	m, _ := route.GetMethods()
	// 	fmt.Println("ROUTE:", t, m)
	// 	return nil
	// })

	return r
}

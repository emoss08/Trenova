package routes

import (
	"github.com/redis/go-redis/v9"
	"trenova/app/middleware"

	"github.com/gorilla/mux"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func InitializeRouter(db *gorm.DB, store *gormstore.Store) *mux.Router {
	r := mux.NewRouter()

	// Server Sent Events
	sseRouter := r.PathPrefix("/sse").Subrouter()
	sseRouter.Use(middleware.SessionMiddleware(store))

	// API routes
	apiRouter := r.PathPrefix("/api").Subrouter()

	// Auth routes
	AuthRoutes(apiRouter, db, store)

	// Public routes
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	protectedRouter := apiRouter.NewRoute().Subrouter()
	protectedRouter.Use(middleware.SessionMiddleware(store))
	protectedRouter.Use(middleware.IdempotencyMiddleware(redisClient))
	protectedRouter.Use(middleware.BasicLoggingMiddleware)

	OrganizationRoutes(protectedRouter, db)           // Organization Routes
	AccountingControlRoutes(protectedRouter, db)      // AccountingControl routes
	BillingControlRoutes(protectedRouter, db)         // BillingControl routes
	InvoiceControlRoutes(protectedRouter, db)         // InvoiceControl routes
	DispatchControlRoutes(protectedRouter, db)        // DispatchControl routes
	ShipmentControlRoutes(protectedRouter, db)        // ShipmentControl routes
	FeasibilityToolControlRoutes(protectedRouter, db) // FeasibilityToolControl routes
	RouteControlRoutes(protectedRouter, db)           // RouteControl routes
	RevenueCodeRoutes(protectedRouter, db)            // RevenueCode routes
	UserRoutes(protectedRouter, db)                   // User routes
	UsersRoutes(protectedRouter, db)                  // Users routes
	EmailProfileRoutes(protectedRouter, db)           // EmailProfile routes

	return r
}

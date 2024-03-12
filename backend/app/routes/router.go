package routes

import (
	"log"

	"trenova/app/middleware"
	"trenova/utils"

	"github.com/gorilla/mux"
	"github.com/henvic/httpretty"
	"github.com/redis/go-redis/v9"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func InitializeRouter(db *gorm.DB, store *gormstore.Store) *mux.Router {
	r := mux.NewRouter()

	validator, err := utils.NewValidator()
	if err != nil {
		log.Fatalf("Error initializing validator: %v", err)
	}

	logger := &httpretty.Logger{
		Time:           true,
		TLS:            true,
		RequestHeader:  true,
		RequestBody:    true,
		ResponseHeader: true,
		ResponseBody:   true,
		Colors:         true, // erase line if you don't like colors
	}

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

	protectedRouter.Use(logger.Middleware)

	OrganizationRoutes(protectedRouter, db, validator)           // Organization Routes
	AccountingControlRoutes(protectedRouter, db, validator)      // AccountingControl routes
	BillingControlRoutes(protectedRouter, db, validator)         // BillingControl routes
	InvoiceControlRoutes(protectedRouter, db, validator)         // InvoiceControl routes
	DispatchControlRoutes(protectedRouter, db, validator)        // DispatchControl routes
	ShipmentControlRoutes(protectedRouter, db, validator)        // ShipmentControl routes
	FeasibilityToolControlRoutes(protectedRouter, db, validator) // FeasibilityToolControl routes
	RouteControlRoutes(protectedRouter, db, validator)           // RouteControl routes
	RevenueCodeRoutes(protectedRouter, db, validator)            // RevenueCode routes
	UserRoutes(protectedRouter, db, validator)                   // User routes
	UsersRoutes(protectedRouter, db, validator)                  // Users routes
	EmailProfileRoutes(protectedRouter, db, validator)           // EmailProfile routes

	return r
}

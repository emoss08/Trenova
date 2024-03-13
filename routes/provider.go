package routes

import (
	"github.com/gorilla/mux"
	"github.com/henvic/httpretty"
)

func InitializeRouter() *mux.Router {
	r := mux.NewRouter()

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
	// sseRouter := r.PathPrefix("/sse").Subrouter()
	// sseRouter.Use(middleware.SessionMiddleware(store))

	// API routes
	apiRouter := r.PathPrefix("/api").Subrouter()

	// // Public routes
	// redisClient := redis.NewClient(&redis.Options{
	// 	Addr: "localhost:6379",
	// })

	protectedRouter := apiRouter.NewRoute().Subrouter()
	// protectedRouter.Use(middleware.SessionMiddleware(store))
	// protectedRouter.Use(middleware.IdempotencyMiddleware(redisClient))
	// protectedRouter.Use(middleware.BasicLoggingMiddleware)

	protectedRouter.Use(logger.Middleware)

	// Reigster Business Unit Routes
	registerBusinessUnitRouter(protectedRouter)

	// Register Billing Control Routes
	registerBillingControlRouter(protectedRouter)

	// Register Invoice Control Routes
	registerInvoiceControlRouter(protectedRouter)

	return r
}

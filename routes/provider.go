package routes

import (
	"github.com/emoss08/trenova/middleware"

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
		Colors:         true,
	}

	// Server Sent Events
	// sseRouter := r.PathPrefix("/sse").Subrouter()
	// sseRouter.Use(middleware.SessionMiddleware(store))

	// API routes
	apiRouter := r.PathPrefix("/api").Subrouter()

	// Register the auth routes
	registerAuthRoutes(apiRouter)

	// Protected routes
	protectedRouter := apiRouter.NewRoute().Subrouter()
	protectedRouter.Use(middleware.SessionMiddleware)
	protectedRouter.Use(middleware.IdempotencyMiddleware)
	protectedRouter.Use(logger.Middleware)

	// Reigster Business Unit Routes
	registerBusinessUnitRouter(protectedRouter)

	// Register Billing Control Routes
	registerBillingControlRouter(protectedRouter)

	// Register Invoice Control Routes
	registerInvoiceControlRouter(protectedRouter)

	// Register User Routes
	registerUserRoutes(protectedRouter)

	return r
}

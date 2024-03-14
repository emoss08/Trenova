package routes

import (
	"fmt"

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

	// Register Organization Routes
	registerOrganizationRoutes(protectedRouter)

	// Register Billing Control Routes
	registerBillingControlRouter(protectedRouter)

	// Register Invoice Control Routes
	registerInvoiceControlRouter(protectedRouter)

	// Register User Routes
	registerUserRoutes(protectedRouter)

	// Register User Favorites Routes
	registerUserFavoritesRoutes(protectedRouter)

	// Walk the router and print out all of the available routes
	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		tpl, err1 := route.GetPathTemplate()
		met, err2 := route.GetMethods()
		fmt.Println(tpl, err1, met, err2)
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}

	return r
}

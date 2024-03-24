package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerRouteControlRouter(r chi.Router) {
	r.Route("/route-control", func(r chi.Router) {
		r.Get("/", controllers.GetRouteControl)
		r.Put("/{routeControlID}", controllers.UpdateRouteControl)
	})
}

package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerLocationRouter(r chi.Router) {
	r.Route("/locations", func(r chi.Router) {
		r.Get("/", controllers.GetLocations)
		r.Post("/", controllers.CreateLocation)
		r.Put("/{locationID}", controllers.UpdateLocation)
	})
}

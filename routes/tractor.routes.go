package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerTractorRouter(r chi.Router) {
	r.Route("/tractors", func(r chi.Router) {
		r.Get("/", controllers.GetTractors)
		r.Post("/", controllers.CreateTractor)
		r.Put("/{tractorID}", controllers.UpdateTractor)
	})
}

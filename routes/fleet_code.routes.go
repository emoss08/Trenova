package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerFleetCodeRouter(r chi.Router) {
	r.Route("/fleet-codes", func(r chi.Router) {
		r.Get("/", controllers.GetFleetCodes)
		r.Post("/", controllers.CreateFleetCode)
		r.Put("/{fleetCodeID}", controllers.UpdateFleetCode)
	})
}

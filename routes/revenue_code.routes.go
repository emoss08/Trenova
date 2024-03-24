package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerRevenueCodeRouter(r chi.Router) {
	r.Route("/revenue-codes", func(r chi.Router) {
		r.Get("/", controllers.GetRevenueCodes)
		r.Post("/", controllers.CreateRevenueCode)
		r.Put("/{revenueCodeID}", controllers.UpdateRevenueCode)
	})
}

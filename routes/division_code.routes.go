package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerDivisionCodeRouter(r chi.Router) {
	r.Route("/division-codes", func(r chi.Router) {
		r.Get("/", controllers.GetDivisionCodes)
		r.Post("/", controllers.CreateDivisionCode)
		r.Put("/{divisionCodeID}", controllers.UpdateDivisionCode)
	})
}

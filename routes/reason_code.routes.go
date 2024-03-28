package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerReasonCodeRouter(r chi.Router) {
	r.Route("/reason-codes", func(r chi.Router) {
		r.Get("/", controllers.GetReasonCode)
		r.Post("/", controllers.CreateReasonCode)
		r.Put("/{reasonCodeID}", controllers.UpdateReasonCode)
	})
}

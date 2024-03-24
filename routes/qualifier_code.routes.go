package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerQualifierCodeRouter(r chi.Router) {
	r.Route("/qualifier-codes", func(r chi.Router) {
		r.Get("/", controllers.GetQualifierCodes)
		r.Post("/", controllers.CreateQualifierCode)
		r.Put("/{qualifierCodeID}", controllers.UpdateQualifierCode)
	})
}

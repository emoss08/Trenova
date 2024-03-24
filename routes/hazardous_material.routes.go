package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerHazardousMaterialRouter(r chi.Router) {
	r.Route("/hazardous-materials", func(r chi.Router) {
		r.Get("/", controllers.GetHazardousMaterial)
		r.Post("/", controllers.CreateHazardousMaterial)
		r.Put("/{hazmatID}", controllers.UpdateHazardousMaterial)
	})
}

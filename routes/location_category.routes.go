package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerLocationCategoryRouter(r chi.Router) {
	r.Route("/location-categories", func(r chi.Router) {
		r.Get("/", controllers.GetLocationCategories)
		r.Post("/", controllers.CreateLocationCategory)
		r.Put("/{locationCategoryID}", controllers.UpdateLocationCategory)
	})
}

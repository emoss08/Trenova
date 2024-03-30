package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerTagRouter(r chi.Router) {
	r.Route("/tags", func(r chi.Router) {
		r.Get("/", controllers.GetTags)
		r.Post("/", controllers.CreateTag)
		r.Put("/{tagID}", controllers.UpdateTag)
	})
}

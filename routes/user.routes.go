package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerUserRoutes(r chi.Router) {
	r.Route("/me", func(r chi.Router) {
		r.Get("/", controllers.GetAuthenticatedUser)
	})
}

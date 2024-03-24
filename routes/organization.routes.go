package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerOrganizationRoutes(r chi.Router) {
	r.Route("/organization", func(r chi.Router) {
		r.Get("/me", controllers.GetUserOrganization)
	})
}

package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerOrganizationFeatureFlagRoutes(r chi.Router) {
	r.Route("/organization-feature-flags", func(r chi.Router) {
		r.Get("/", controllers.GetOrganizationFeatureFlags)
	})
}

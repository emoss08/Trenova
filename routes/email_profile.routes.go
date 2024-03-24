package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerEmailProfileRouter(r chi.Router) {
	r.Route("/email-profiles", func(r chi.Router) {
		r.Get("/", controllers.GetEmailProfiles)
		r.Post("/", controllers.CreateEmailProfile)
		r.Put("/{emailProfileID}", controllers.UpdateEmailProfile)
	})
}

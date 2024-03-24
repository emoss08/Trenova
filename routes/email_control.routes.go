package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerEmailControlRoutes(r chi.Router) {
	r.Route("/email-control", func(r chi.Router) {
		r.Get("/", controllers.GetEmailControl)
		r.Put("/{emailControlID}", controllers.UpdateEmailControl)
	})
}

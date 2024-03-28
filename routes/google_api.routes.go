package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerGoogleAPIRouter(r chi.Router) {
	r.Route("/google-api", func(r chi.Router) {
		r.Get("/", controllers.GetGoogleAPI)
		r.Put("/{googleAPIID}", controllers.UpdateGoogleAPI)
	})
}

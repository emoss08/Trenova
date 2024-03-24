package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerUsStateRouter(r chi.Router) {
	r.Route("/us-states", func(r chi.Router) {
		r.Get("/", controllers.GetUsStates)
	})
}

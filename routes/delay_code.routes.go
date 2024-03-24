package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerDelayCodeRouter(r chi.Router) {
	r.Route("/delay-codes", func(r chi.Router) {
		r.Get("/", controllers.GetDelayCodes)
		r.Post("/", controllers.CreateDelayCode)
		r.Put("/{delayCodeID}", controllers.UpdateDelayCode)
	})
}

package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerChargeTypeRouter(r chi.Router) {
	r.Route("/charge-types", func(r chi.Router) {
		r.Get("/", controllers.GetChargeTypes)
		r.Post("/", controllers.CreateChargeType)
		r.Put("/{chargeTypeID}", controllers.UpdateChargeType)
	})
}

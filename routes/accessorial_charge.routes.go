package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerAccessorialChargeRouter(r chi.Router) {
	r.Route("/accessorial-charges", func(r chi.Router) {
		r.Get("/", controllers.GetAccessorialCharge)
		r.Post("/", controllers.CreateAccessorialCharge)
		r.Put("/{accessorialChargeID}", controllers.UpdateAccessorialCharge)
	})
}

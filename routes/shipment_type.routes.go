package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerShipmentTypeRouter(r chi.Router) {
	r.Route("/shipment-types", func(r chi.Router) {
		r.Get("/", controllers.GetShipmentTypes)
		r.Post("/", controllers.CreateShipmentType)
		r.Put("/{shipTypeID}", controllers.UpdateShipmentType)
	})
}

package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerShipmentControlRouter(r chi.Router) {
	r.Route("/shipment-control", func(r chi.Router) {
		r.Get("/", controllers.GetShipmentControl)
		r.Put("/{shipmentControlID}", controllers.UpdateShipmentControl)
	})
}

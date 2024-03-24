package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerEquipmentManufacturerRouter(r chi.Router) {
	r.Route("/equipment-manufacturers", func(r chi.Router) {
		r.Get("/", controllers.GetEquipmentManufacturer)
		r.Post("/", controllers.CreateEquipmentManufacturer)
		r.Put("/{equipManuID}", controllers.UpdateEquipmentManfacturer)
	})
}

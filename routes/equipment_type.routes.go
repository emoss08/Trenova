package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerEquipmentTypeRouter(r chi.Router) {
	r.Route("/equipment-types", func(r chi.Router) {
		r.Get("/", controllers.GetEquipmentTypes)
		r.Post("/", controllers.CreateEquipmentType)
		r.Put("/{equipTypeID}", controllers.UpdateEquipmentType)
	})
}

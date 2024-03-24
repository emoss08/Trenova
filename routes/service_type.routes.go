package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerServiceTypeRouter(r chi.Router) {
	r.Route("/service-types", func(r chi.Router) {
		r.Get("/", controllers.GetServiceTypes)
		r.Post("/", controllers.CreateServiceType)
		r.Put("/{serviceTypeID}", controllers.UpdateServiceType)
	})
}

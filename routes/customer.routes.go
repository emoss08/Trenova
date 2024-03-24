package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerCustomerRouter(r chi.Router) {
	r.Route("/customers", func(r chi.Router) {
		r.Get("/", controllers.GetCustomers)
		r.Post("/", controllers.CreateCustomer)
		r.Put("/{customerID}", controllers.UpdateCustomer)
	})
}

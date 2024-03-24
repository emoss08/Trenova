package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerInvoiceControlRouter(r chi.Router) {
	r.Route("/invoice-control", func(r chi.Router) {
		r.Get("/", controllers.GetInvoiceControl)
		r.Put("/{invoiceControlID}", controllers.UpdateInvoiceControl)
	})
}

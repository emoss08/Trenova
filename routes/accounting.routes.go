package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerBillingControlRouter(r chi.Router) {
	r.Route("/billing-control", func(r chi.Router) {
		r.Get("/", controllers.GetBillingControl)
		r.Put("/{billingControlID}", controllers.UpdateBillingControl)
	})
}

func registerAccountingControlRouter(r chi.Router) {
	r.Route("/accounting-control", func(r chi.Router) {
		r.Get("/", controllers.GetAccountingControl)
		r.Put("/{accountingControlID}", controllers.UpdateAccountingControl)
	})
}

package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerDispatchControlRouter(r chi.Router) {
	r.Route("/dispatch-control", func(r chi.Router) {
		r.Get("/", controllers.GetDispatchControl)
		r.Put("/{dispatchControlID}", controllers.UpdateDispatchControl)
	})
}

func registerFeasibilityControlRouter(r chi.Router) {
	r.Route("/feasibility-tool-control", func(r chi.Router) {
		r.Get("/", controllers.GetFeasibilityToolControl)
		r.Put("/{feasibilityToolControlID}", controllers.UpdateFeasibilityToolControl)
	})
}

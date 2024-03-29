package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerHazmatSegRuleRouter(r chi.Router) {
	r.Route("/hazardous-material-segregations", func(r chi.Router) {
		r.Get("/", controllers.GetHazmatSegRules)
		r.Post("/", controllers.CreateHazmatSegRule)
		r.Put("/{hazmatSegRuleID}", controllers.UpdateHazmatSegRule)
	})
}

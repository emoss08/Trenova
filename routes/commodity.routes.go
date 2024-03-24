package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerCommodityRoutes(r chi.Router) {
	r.Route("/commodities", func(r chi.Router) {
		r.Get("/", controllers.GetCommodities)
		r.Post("/", controllers.CreateCommodity)
		r.Put("/{commodityID}", controllers.UpdateCommodity)
	})
}

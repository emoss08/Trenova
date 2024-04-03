package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerTrailerRouter(r chi.Router) {
	r.Route("/trailers", func(r chi.Router) {
		r.Get("/", controllers.GetTrailers)
		r.Post("/", controllers.CreateTrailer)
		r.Put("/{trailerID}", controllers.UpdateTrailer)
	})
}

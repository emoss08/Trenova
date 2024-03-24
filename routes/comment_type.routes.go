package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerCommentTypeRouter(r chi.Router) {
	r.Route("/comment-types", func(r chi.Router) {
		r.Get("/", controllers.GetCommentTypes)
		r.Post("/", controllers.CreateCommentType)
		r.Put("/{commentTypeID}", controllers.UpdateCommentType)
	})
}

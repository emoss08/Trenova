package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerDocumentClassificationRouter(r chi.Router) {
	r.Route("/document-classifications", func(r chi.Router) {
		r.Get("/", controllers.GetDocumentClassifications)
		r.Post("/", controllers.CreateDocumentClassification)
		r.Put("/{docClassID}", controllers.UpdateDocumentClassification)
	})
}

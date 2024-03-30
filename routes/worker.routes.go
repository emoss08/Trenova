package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerWorkerRouter(r chi.Router) {
	r.Route("/workers", func(r chi.Router) {
		r.Get("/", controllers.GetWorkers)
		r.Post("/", controllers.CreateWorker)
		r.Put("/{workerID}", controllers.UpdateWorker)
	})
}

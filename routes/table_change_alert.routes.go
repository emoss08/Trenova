package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerTableChangeAlertRouter(r chi.Router) {
	r.Route("/table-change-alerts", func(r chi.Router) {
		r.Get("/", controllers.GetTableChangeAlerts)
		r.Post("/", controllers.CreateTableChangeALert)
		r.Put("/{tableChangeAlertID}", controllers.UpdateTableChangeAlert)
		r.Get("/table-names", controllers.GetTableNames)
		r.Get("/topic-names", controllers.GetTopicNames)
	})
}

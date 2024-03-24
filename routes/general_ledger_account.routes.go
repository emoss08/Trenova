package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

func registerGLAccountRoutes(r chi.Router) {
	r.Route("/general-ledger-accounts", func(r chi.Router) {
		r.Get("/", controllers.GetGeneralLedgerAccounts)
		r.Post("/", controllers.CreateGeneralLedgerAccount)
		r.Put("/{glAccountID}", controllers.UpdateGeneralLedgerAccount)
	})
}

package routes

import (
	handlers "github.com/emoss08/trenova/apis"
	"github.com/emoss08/trenova/models"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func RevenueCodeRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	revCodeService := &handlers.RevenueCodeHandler{DB: db}

	revCodeRouter := r.PathPrefix("/revenue-codes").Subrouter()

	revCodeRouter.HandleFunc("/", services.GetEntityHandler[models.RevenueCode](revCodeService)).Methods("GET")
	revCodeRouter.HandleFunc("/{entityID}/", services.GetEntityByIDHandler[models.RevenueCode](revCodeService)).Methods("GET")
	revCodeRouter.HandleFunc("/", services.CreateEntityHandler[models.RevenueCode](revCodeService, validator)).Methods("POST")
	revCodeRouter.HandleFunc("/{entityID}/", services.UpdateEntityHandler[models.RevenueCode](revCodeService, validator)).Methods("PUT")
}

func AccountingControlRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	acRouter := r.PathPrefix("/accounting-control").Subrouter()

	acRouter.HandleFunc("/", handlers.GetAccountingControl(db)).Methods("GET")
	acRouter.HandleFunc("/", handlers.UpdateAccountingControl(db, validator)).Methods("PUT")
}

func BillingControlRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	bcRouter := r.PathPrefix("/billing-control").Subrouter()

	bcRouter.HandleFunc("/", handlers.GetBillingControl(db)).Methods("GET")
	bcRouter.HandleFunc("/", handlers.UpdateBillingControl(db, validator)).Methods("PUT")
}

func InvoiceControlRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	icRouter := r.PathPrefix("/invoice-control").Subrouter()

	icRouter.HandleFunc("/", handlers.GetInvoiceControl(db)).Methods("GET")
	icRouter.HandleFunc("/", handlers.UpdateInvoiceControl(db, validator)).Methods("PUT")
}

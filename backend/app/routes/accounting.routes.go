package routes

import (
	"trenova/app/handlers"
	"trenova/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func RevenueCodeRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	revCodeRouter := r.PathPrefix("/revenue-codes").Subrouter()

	revCodeRouter.HandleFunc("/", handlers.GetRevenueCodes(db)).Methods("GET")
	revCodeRouter.HandleFunc("/", handlers.CreateRevenueCode(db, validator)).Methods("POST")
	revCodeRouter.HandleFunc("/{revenueCodeID}/", handlers.GetRevenueCodeByID(db)).Methods("GET")
	revCodeRouter.HandleFunc("/{revenueCodeID}/", handlers.UpdateRevenueCode(db, validator)).Methods("PUT")
}

func AccountingControlRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	acRouter := r.PathPrefix("/accounting-control").Subrouter()

	acRouter.HandleFunc("/", handlers.GetAccountingControl(db)).Methods("GET")
	acRouter.HandleFunc("/", handlers.UpdateAccountingControl(db, validator)).Methods("PUT")
}

func BillingControlRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	bcRouter := r.PathPrefix("/billing-control").Subrouter()

	bcRouter.HandleFunc("/", handlers.GetBillingControl(db)).Methods("GET")
	bcRouter.HandleFunc("/", handlers.UpdateBillingControl(db, validator)).Methods("PUT")
}

func InvoiceControlRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	icRouter := r.PathPrefix("/invoice-control").Subrouter()

	icRouter.HandleFunc("/", handlers.GetInvoiceControl(db)).Methods("GET")
	icRouter.HandleFunc("/", handlers.UpdateInvoiceControl(db, validator)).Methods("PUT")
}

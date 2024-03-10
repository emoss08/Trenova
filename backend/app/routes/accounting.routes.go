package routes

import (
	"trenova/app/handlers"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func RevenueCodeRoutes(r *mux.Router, db *gorm.DB) {
	revCodeRouter := r.PathPrefix("/revenue-codes").Subrouter()

	revCodeRouter.HandleFunc("/", handlers.GetRevenueCodes(db)).Methods("GET")
	revCodeRouter.HandleFunc("/", handlers.CreateRevenueCode(db)).Methods("POST")
	revCodeRouter.HandleFunc("/{revenueCodeID}/", handlers.GetRevenueCodeByID(db)).Methods("GET")
	revCodeRouter.HandleFunc("/{revenueCodeID}/", handlers.UpdateRevenueCode(db)).Methods("PUT")
}

package routes

import (
	"trenova-go-backend/app/handlers"

	"github.com/gorilla/mux"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func RevenueCodeRoutes(r *mux.Router, db *gorm.DB, store *gormstore.Store) {
	revCodeRouter := r.PathPrefix("/revenue-codes").Subrouter()

	revCodeRouter.HandleFunc("/", handlers.GetRevenueCodes(db, store)).Methods("GET")
	revCodeRouter.HandleFunc("/", handlers.CreateRevenueCode(db)).Methods("POST")
	revCodeRouter.HandleFunc("/{revenueCodeID}/", handlers.GetRevenueCodeByID(db)).Methods("GET")
	revCodeRouter.HandleFunc("/{revenueCodeID}/", handlers.UpdateRevenueCode(db)).Methods("PUT")
}

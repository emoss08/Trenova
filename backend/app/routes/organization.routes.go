package routes

import (
	"trenova-go-backend/app/handlers"

	"github.com/gorilla/mux"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func OrganizationRoutes(r *mux.Router, db *gorm.DB, store *gormstore.Store) {
	or := r.PathPrefix("/organization").Subrouter()

	or.HandleFunc("/me/", handlers.GetOrganization(db, store)).Methods("GET", "OPTIONS")
}

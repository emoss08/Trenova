package routes

import (
	"trenova/app/handlers"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func OrganizationRoutes(r *mux.Router, db *gorm.DB) {
	or := r.PathPrefix("/organization").Subrouter()

	or.HandleFunc("/me/", handlers.GetOrganization(db)).Methods("GET")
	or.HandleFunc("/", handlers.UpdateOrganization(db)).Methods("PUT")
}

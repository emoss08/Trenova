package routes

import (
	"trenova/app/handlers"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func OrganizationRoutes(r *mux.Router, db *gorm.DB) {
	orgRouter := r.PathPrefix("/organization").Subrouter()

	orgRouter.HandleFunc("/me/", handlers.GetOrganization(db)).Methods("GET")
	orgRouter.HandleFunc("/", handlers.UpdateOrganization(db)).Methods("PUT")
}

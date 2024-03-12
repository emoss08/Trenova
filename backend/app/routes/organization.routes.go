package routes

import (
	"trenova/app/handlers"
	"trenova/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func OrganizationRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	orgRouter := r.PathPrefix("/organization").Subrouter()

	orgRouter.HandleFunc("/me/", handlers.GetOrganization(db)).Methods("GET")
	orgRouter.HandleFunc("/", handlers.UpdateOrganization(db, validator)).Methods("PUT")
}

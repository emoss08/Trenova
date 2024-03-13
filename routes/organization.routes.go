package routes

import (
	handlers "github.com/emoss08/trenova/apis"
	"github.com/emoss08/trenova/tools"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func OrganizationRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	orgRouter := r.PathPrefix("/organization").Subrouter()

	orgRouter.HandleFunc("/me/", handlers.GetOrganization(db)).Methods("GET")
	orgRouter.HandleFunc("/", handlers.UpdateOrganization(db, validator)).Methods("PUT")
}

package routes

import (
	"trenova/app/handlers"
	"trenova/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func DispatchControlRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	dcRouter := r.PathPrefix("/dispatch-control").Subrouter()

	dcRouter.HandleFunc("/", handlers.GetDispatchControl(db)).Methods("GET")
	dcRouter.HandleFunc("/", handlers.UpdateDispatchControl(db, validator)).Methods("PUT")
}

func FeasibilityToolControlRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	ftc := r.PathPrefix("/feasibility-control").Subrouter()

	ftc.HandleFunc("/", handlers.GetFeasibilityToolControl(db)).Methods("GET")
	ftc.HandleFunc("/", handlers.UpdateFeasibilityToolControl(db, validator)).Methods("PUT")
}

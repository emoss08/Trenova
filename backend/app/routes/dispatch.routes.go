package routes

import (
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"trenova/app/handlers"
)

func DispatchControlRoutes(r *mux.Router, db *gorm.DB) {
	dcRouter := r.PathPrefix("/dispatch-control").Subrouter()

	dcRouter.HandleFunc("/", handlers.GetDispatchControl(db)).Methods("GET")
	dcRouter.HandleFunc("/", handlers.UpdateDispatchControl(db)).Methods("PUT")
}

func FeasibilityToolControlRoutes(r *mux.Router, db *gorm.DB) {
	ftc := r.PathPrefix("/feasibility-control").Subrouter()

	ftc.HandleFunc("/", handlers.GetFeasibilityToolControl(db)).Methods("GET")
	ftc.HandleFunc("/", handlers.UpdateFeasibilityToolControl(db)).Methods("PUT")
}

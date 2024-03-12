package routes

import (
	"trenova/app/handlers"
	"trenova/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func ShipmentControlRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	scRouter := r.PathPrefix("/shipment-control").Subrouter()

	scRouter.HandleFunc("/", handlers.GetShipmentControl(db)).Methods("GET")
	scRouter.HandleFunc("/", handlers.UpdateShipmentControl(db, validator)).Methods("PUT")
}

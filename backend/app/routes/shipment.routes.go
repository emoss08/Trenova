package routes

import (
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"trenova/app/handlers"
)

func ShipmentControlRoutes(r *mux.Router, db *gorm.DB) {
	scRouter := r.PathPrefix("/shipment-control").Subrouter()

	scRouter.HandleFunc("/", handlers.GetShipmentControl(db)).Methods("GET")
	scRouter.HandleFunc("/", handlers.UpdateShipmentControl(db)).Methods("PUT")
}

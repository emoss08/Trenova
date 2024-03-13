package routes

import (
	handlers "github.com/emoss08/trenova/apis"
	"github.com/emoss08/trenova/tools"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func ShipmentControlRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	scRouter := r.PathPrefix("/shipment-control").Subrouter()

	scRouter.HandleFunc("/", handlers.GetShipmentControl(db)).Methods("GET")
	scRouter.HandleFunc("/", handlers.UpdateShipmentControl(db, validator)).Methods("PUT")
}

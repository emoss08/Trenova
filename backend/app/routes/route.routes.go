package routes

import (
	"trenova/app/handlers"
	"trenova/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func RouteControlRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	routeControlRouter := r.PathPrefix("/route-control").Subrouter()

	routeControlRouter.HandleFunc("/", handlers.GetRouteControl(db)).Methods("GET")
	routeControlRouter.HandleFunc("/", handlers.UpdateRouteControl(db, validator)).Methods("PUT")
}

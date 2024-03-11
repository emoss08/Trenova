package routes

import (
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"trenova/app/handlers"
)

func RouteControlRoutes(r *mux.Router, db *gorm.DB) {
	routeControlRouter := r.PathPrefix("/route-control").Subrouter()

	routeControlRouter.HandleFunc("/", handlers.GetRouteControl(db)).Methods("GET")
	routeControlRouter.HandleFunc("/", handlers.UpdateRouteControl(db)).Methods("PUT")
}

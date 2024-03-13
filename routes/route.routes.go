package routes

import (
	handlers "github.com/emoss08/trenova/apis"
	"github.com/emoss08/trenova/tools"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func RouteControlRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	routeControlRouter := r.PathPrefix("/route-control").Subrouter()

	routeControlRouter.HandleFunc("/", handlers.GetRouteControl(db)).Methods("GET")
	routeControlRouter.HandleFunc("/", handlers.UpdateRouteControl(db, validator)).Methods("PUT")
}

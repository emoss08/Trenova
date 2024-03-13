package routes

import (
	"net/http"

	"github.com/emoss08/trenova/controllers"
	"github.com/gorilla/mux"
)

func registerBusinessUnitRouter(r *mux.Router) {
	buRouter := r.PathPrefix("/business-units").Subrouter()

	buRouter.HandleFunc("/", controllers.GetBusinessUnits).Methods(http.MethodGet)
}

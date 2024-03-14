package routes

import (
	"net/http"

	"github.com/emoss08/trenova/controllers"
	"github.com/gorilla/mux"
)

func registerOrganizationRoutes(r *mux.Router) {
	bcRouter := r.PathPrefix("/organization").Subrouter()

	bcRouter.HandleFunc("/me/", controllers.GetUserOrganization).Methods(http.MethodGet)
}

package routes

import (
	"net/http"

	"github.com/emoss08/trenova/controllers"
	"github.com/gorilla/mux"
)

func registerUserRoutes(r *mux.Router) {
	meRouter := r.PathPrefix("/me/").Subrouter()

	meRouter.HandleFunc("/", controllers.GetAuthenticatedUser).Methods(http.MethodGet)
}

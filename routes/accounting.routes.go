package routes

import (
	"net/http"

	"github.com/emoss08/trenova/controllers"
	"github.com/gorilla/mux"
)

func registerBillingControlRouter(r *mux.Router) {
	bcRouter := r.PathPrefix("/billing-control").Subrouter()

	bcRouter.HandleFunc("/", controllers.GetBillingControl).Methods(http.MethodGet)
	bcRouter.HandleFunc("/{billingControlID}/", controllers.UpdateBillingControl).Methods(http.MethodPut, http.MethodPatch)
}

package routes

import (
	"net/http"

	"github.com/emoss08/trenova/controllers"
	"github.com/gorilla/mux"
)

func registerInvoiceControlRouter(r *mux.Router) {
	bcRouter := r.PathPrefix("/invoice-control").Subrouter()

	bcRouter.HandleFunc("/", controllers.GetInvoiceControl).Methods(http.MethodGet)
}

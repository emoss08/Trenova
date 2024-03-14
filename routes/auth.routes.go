package routes

import (
	"net/http"

	"github.com/emoss08/trenova/controllers"
	"github.com/gorilla/mux"
)

func registerAuthRoutes(r *mux.Router) {
	authRouter := r.PathPrefix("/auth").Subrouter()

	authRouter.HandleFunc("/login/", controllers.LoginHandler).Methods(http.MethodPost)
}

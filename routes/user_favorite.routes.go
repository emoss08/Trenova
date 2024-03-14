package routes

import (
	"net/http"

	"github.com/emoss08/trenova/controllers"
	"github.com/gorilla/mux"
)

// RegisterUserFavoritesRoutes registers the user favorites routes
func registerUserFavoritesRoutes(r *mux.Router) {
	ufRouter := r.PathPrefix("/user-favorites").Subrouter()

	// UserFavorite for the currently authenticated user.
	ufRouter.HandleFunc("/me/", controllers.GetUserFavorites).Methods(http.MethodGet)
	ufRouter.HandleFunc("/me/", controllers.CreateUserFavorite).Methods(http.MethodPost)
	ufRouter.HandleFunc("/me/", controllers.DeleteUserFavorite).Methods(http.MethodDelete)
}

package routes

import (
	"github.com/emoss08/trenova/controllers"
	"github.com/go-chi/chi/v5"
)

// RegisterUserFavoritesRoutes registers the user favorites routes.
func registerUserFavoritesRoutes(r chi.Router) {
	r.Route("/user-favorites", func(r chi.Router) {
		r.Get("/", controllers.GetUserFavorites)
		r.Post("/", controllers.CreateUserFavorite)
		r.Delete("/", controllers.DeleteUserFavorite)
	})
}

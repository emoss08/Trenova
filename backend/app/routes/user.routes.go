package routes

import (
	"trenova-go-backend/app/handlers"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func UserRoutes(r *mux.Router, db *gorm.DB) {
	userRouter := r.PathPrefix("/user").Subrouter()

	userRouter.HandleFunc("/me/", handlers.GetAuthenticatedUser(db)).Methods("GET")
	userRouter.HandleFunc("/me/favorites/", handlers.GetUserFavorites(db)).Methods("GET")
	userRouter.HandleFunc("/me/favorites/", handlers.AddUserFavorite(db)).Methods("POST")
	userRouter.HandleFunc("/me/favorites/", handlers.RemoveUserFavorite(db)).Methods("DELETE")
}

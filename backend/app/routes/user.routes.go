package routes

import (
	"trenova/app/handlers"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func UserRoutes(r *mux.Router, db *gorm.DB) {
	userRouter := r.PathPrefix("/me").Subrouter()

	userRouter.HandleFunc("/", handlers.GetAuthenticatedUser(db)).Methods("GET")
	userRouter.HandleFunc("/favorites/", handlers.GetUserFavorites(db)).Methods("GET")
	userRouter.HandleFunc("/favorites/", handlers.AddUserFavorite(db)).Methods("POST")
	userRouter.HandleFunc("/favorites/", handlers.RemoveUserFavorite(db)).Methods("DELETE")
}

func UsersRoutes(r *mux.Router, db *gorm.DB) {
	usersRouter := r.PathPrefix("/users").Subrouter()

	usersRouter.HandleFunc("/{userID}/", handlers.UpdateUser(db)).Methods("PUT", "PATCH")
}

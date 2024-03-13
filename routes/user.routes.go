package routes

import (
	handlers "github.com/emoss08/trenova/apis"
	"github.com/emoss08/trenova/tools"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func UserRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	userRouter := r.PathPrefix("/me").Subrouter()

	userRouter.HandleFunc("/", handlers.GetAuthenticatedUser(db)).Methods("GET")
	userRouter.HandleFunc("/favorites/", handlers.GetUserFavorites(db)).Methods("GET")
	userRouter.HandleFunc("/favorites/", handlers.AddUserFavorite(db, validator)).Methods("POST")
	userRouter.HandleFunc("/favorites/", handlers.RemoveUserFavorite(db, validator)).Methods("DELETE")
}

func UsersRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	usersRouter := r.PathPrefix("/users").Subrouter()

	usersRouter.HandleFunc("/{userID}/", handlers.UpdateUser(db, validator)).Methods("PUT", "PATCH")
}

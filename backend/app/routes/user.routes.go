package routes

import (
	"trenova-go-backend/app/handlers"

	"github.com/gorilla/mux"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func UserRoutes(r *mux.Router, db *gorm.DB, store *gormstore.Store) {
	userRouter := r.PathPrefix("/user").Subrouter()

	userRouter.HandleFunc("/me/", handlers.GetAuthenticatedUser(db, store)).Methods("GET", "OPTIONS")
}

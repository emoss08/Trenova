package routes

import (
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"trenova/app/handlers"
)

func EmailProfileRoutes(r *mux.Router, db *gorm.DB) {
	epRouter := r.PathPrefix("/email-profiles").Subrouter()

	epRouter.HandleFunc("/", handlers.GetEmailProfiles(db)).Methods("GET")
	epRouter.HandleFunc("/", handlers.CreateEmailProfile(db)).Methods("POST")
	epRouter.HandleFunc("/{emailProfileID}/", handlers.UpdateEmailProfile(db)).Methods("PUT")
}

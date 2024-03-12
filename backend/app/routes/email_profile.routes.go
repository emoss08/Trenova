package routes

import (
	"trenova/app/handlers"
	"trenova/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func EmailProfileRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	epRouter := r.PathPrefix("/email-profiles").Subrouter()

	epRouter.HandleFunc("/", handlers.GetEmailProfiles(db)).Methods("GET")
	epRouter.HandleFunc("/", handlers.CreateEmailProfile(db, validator)).Methods("POST")
	epRouter.HandleFunc("/{emailProfileID}/", handlers.UpdateEmailProfile(db, validator)).Methods("PUT")
}

package routes

import (
	"trenova-go-backend/app/handlers"

	"github.com/gorilla/mux"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func AuthRoutes(r *mux.Router, db *gorm.DB, store *gormstore.Store) {
	ar := r.PathPrefix("/auth").Subrouter()

	// Allow OPTIONS method for preflight requests along with POST
	// ar.HandleFunc("/signup", handlers.SignUp(db, store)).Methods("POST", "OPTIONS")
	ar.HandleFunc("/login/", handlers.Login(db, store)).Methods("POST", "OPTIONS")
	ar.HandleFunc("/logout/", handlers.Logout(store)).Methods("POST", "OPTIONS")

}

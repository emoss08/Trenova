package routes

import (
	handlers "github.com/emoss08/trenova/apis"
	"github.com/gorilla/mux"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func AuthRoutes(r *mux.Router, db *gorm.DB, store *gormstore.Store) {
	aRouter := r.PathPrefix("/auth").Subrouter()

	// Allow OPTIONS method for preflight requests along with POST
	// aRouter.HandleFunc("/signup", handlers.SignUp(db, store)).Methods("POST", "OPTIONS")
	aRouter.HandleFunc("/login/", handlers.Login(db, store)).Methods("POST", "OPTIONS")
	aRouter.HandleFunc("/logout/", handlers.Logout(store)).Methods("POST", "OPTIONS")
}

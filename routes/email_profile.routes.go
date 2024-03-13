package routes

import (
	handlers "github.com/emoss08/trenova/apis"
	"github.com/emoss08/trenova/models"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func EmailProfileRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	epService := &handlers.EmailProfileHandler{DB: db}

	epRouter := r.PathPrefix("/email-profiles").Subrouter()

	epRouter.HandleFunc("/", services.GetEntityHandler[models.EmailProfile](epService)).Methods("GET")
	epRouter.HandleFunc("/{entityID}/", services.GetEntityByIDHandler[models.EmailProfile](epService)).Methods("GET")
	epRouter.HandleFunc("/", services.CreateEntityHandler[models.EmailProfile](epService, validator)).Methods("POST")
	epRouter.HandleFunc("/{entityID}/", services.UpdateEntityHandler[models.EmailProfile](epService, validator)).Methods("PUT")
}

package routes

import (
	"trenova/app/handlers"
	"trenova/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func QualifierCodeRoutes(r *mux.Router, db *gorm.DB, validator *utils.Validator) {
	qcRouter := r.PathPrefix("/qualifier-codes").Subrouter()

	qcRouter.HandleFunc("/", handlers.GetQualifierCodes(db)).Methods("GET")
	qcRouter.HandleFunc("/{qualifierCodeID}/", handlers.GetQualifierCodeByID(db)).Methods("GET")
	qcRouter.HandleFunc("/", handlers.CreateQualifierCode(db, validator)).Methods("POST")
	qcRouter.HandleFunc("/{qualifierCodeID}/", handlers.UpdateQualifierCode(db, validator)).Methods("PUT")
}

package routes

import (
	handlers "github.com/emoss08/trenova/apis"
	"github.com/emoss08/trenova/models"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func QualifierCodeRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	qCodeService := &handlers.QualifierCodeHandler{DB: db}

	qcRouter := r.PathPrefix("/qualifier-codes").Subrouter()

	qcRouter.HandleFunc("/", services.GetEntityHandler[models.QualifierCode](qCodeService)).Methods("GET")
	qcRouter.HandleFunc("/{entityID}/", services.GetEntityByIDHandler[models.QualifierCode](qCodeService)).Methods("GET")
	qcRouter.HandleFunc("/", services.CreateEntityHandler[models.QualifierCode](qCodeService, validator)).Methods("POST")
	qcRouter.HandleFunc("/{entityID}/", services.UpdateEntityHandler[models.QualifierCode](qCodeService, validator)).Methods("PUT")
}

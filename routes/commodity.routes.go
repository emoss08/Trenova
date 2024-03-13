package routes

import (
	handlers "github.com/emoss08/trenova/apis"
	"github.com/emoss08/trenova/models"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func CommodityRoutes(r *mux.Router, db *gorm.DB, validator *tools.Validator) {
	commodityService := &handlers.CommodityHandler{DB: db}

	cmRouter := r.PathPrefix("/commodities").Subrouter()

	cmRouter.HandleFunc("/", services.GetEntityHandler[models.Commodity](commodityService)).Methods("GET")
	cmRouter.HandleFunc("/{entityID}/", services.GetEntityByIDHandler[models.Commodity](commodityService)).Methods("GET")
	cmRouter.HandleFunc("/", services.CreateEntityHandler[models.Commodity](commodityService, validator)).Methods("POST")
	cmRouter.HandleFunc("/{entityID}/", services.UpdateEntityHandler[models.Commodity](commodityService, validator)).Methods("PUT")
}

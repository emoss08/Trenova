package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetShipmentControl gets the shipment control settings for an organization.
func GetShipmentControl(w http.ResponseWriter, r *http.Request) {
	orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
	buID, buOK := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

	if !ok || !buOK {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorResponse{
			Type: "internalError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "internalError",
					Detail: "Organization ID or Business Unit ID not found in the request context",
					Attr:   "orgID, buID",
				},
			},
		})

		return
	}

	shipmentControl, err := services.NewShipmentControlOps(r.Context()).GetShipmentControl(orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, shipmentControl)
}

// UpdateShipmentControl updates the shipment control settings for an organization.
func UpdateShipmentControl(w http.ResponseWriter, r *http.Request) {
	shipmentControlID := chi.URLParam(r, "shipmentControlID")
	if shipmentControlID == "" {
		return
	}

	var sControlData ent.ShipmentControl

	if err := tools.ParseBodyAndValidate(w, r, &sControlData); err != nil {
		return
	}

	sControlData.ID = uuid.MustParse(shipmentControlID)

	invoiceControl, err := services.NewShipmentControlOps(r.Context()).UpdateShipmentControl(sControlData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, invoiceControl)
}

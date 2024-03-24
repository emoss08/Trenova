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

// GetShipmentTypes gets the shipment types for an organization.
func GetShipmentTypes(w http.ResponseWriter, r *http.Request) {
	offset, limit, err := tools.PaginationParams(r)
	if err != nil {
		tools.ResponseWithError(w, http.StatusBadRequest, types.ValidationErrorResponse{
			Type: "invalidRequest",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "invalidRequest",
					Detail: err.Error(),
					Attr:   "offset, limit",
				},
			},
		})

		return
	}

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

	shipmentTypes, count, err := services.NewShipmentTypeOps(r.Context()).GetShipmentTypes(limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  shipmentTypes,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateShipmentType creates a new shipment type.
func CreateShipmentType(w http.ResponseWriter, r *http.Request) {
	var newShipmentType ent.ShipmentType

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

	newShipmentType.BusinessUnitID = buID
	newShipmentType.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newShipmentType); err != nil {
		return
	}

	createEquipType, err := services.NewShipmentTypeOps(r.Context()).CreateShipmentType(newShipmentType)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createEquipType)
}

// UpdateShipmentType updates a shipment type by ID.
func UpdateShipmentType(w http.ResponseWriter, r *http.Request) {
	shipTypeID := chi.URLParam(r, "shipTypeID")
	if shipTypeID == "" {
		return
	}

	var shipmentTypeData ent.ShipmentType

	if err := tools.ParseBodyAndValidate(w, r, &shipmentTypeData); err != nil {
		return
	}

	shipmentTypeData.ID = uuid.MustParse(shipTypeID)

	shipmentType, err := services.NewShipmentTypeOps(r.Context()).UpdateShipmentType(shipmentTypeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, shipmentType)
}

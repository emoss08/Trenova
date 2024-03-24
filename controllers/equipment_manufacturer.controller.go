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

// GetEquipmentManufacturer gets the equipment manufacturers for an organization.
func GetEquipmentManufacturer(w http.ResponseWriter, r *http.Request) {
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

	equipmentManufacturers, count, err := services.NewEquipmentManufactuerOps(r.Context()).GetEquipmentManufacturers(limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  equipmentManufacturers,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateEquipmentManufacturer creates a new equipment manfuacturer.
func CreateEquipmentManufacturer(w http.ResponseWriter, r *http.Request) {
	var newEquipManu ent.EquipmentManufactuer

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

	newEquipManu.BusinessUnitID = buID
	newEquipManu.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newEquipManu); err != nil {
		return
	}

	createEquipManu, err := services.NewEquipmentManufactuerOps(r.Context()).CreateEquipmentManufacturer(newEquipManu)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createEquipManu)
}

// UpdateEquipmentManfacturer updates a equipment manfuacturer by ID.
func UpdateEquipmentManfacturer(w http.ResponseWriter, r *http.Request) {
	equipManuID := chi.URLParam(r, "equipManuID")
	if equipManuID == "" {
		return
	}

	var equipManuData ent.EquipmentManufactuer

	if err := tools.ParseBodyAndValidate(w, r, &equipManuData); err != nil {
		return
	}

	equipManuData.ID = uuid.MustParse(equipManuID)

	equipmentManufacturer, err := services.NewEquipmentManufactuerOps(r.Context()).UpdateEquipmentManufacturer(equipManuData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, equipmentManufacturer)
}

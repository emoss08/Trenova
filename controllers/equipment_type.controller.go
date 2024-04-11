package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetEquipmentTypes gets the equipment types for an organization.
func GetEquipmentTypes(w http.ResponseWriter, r *http.Request) {
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

	equipmentTypes, count, err := services.NewEquipmentTypeOps().GetEquipmentTypes(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  equipmentTypes,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateEquipmentType creates a new equipment type.
func CreateEquipmentType(w http.ResponseWriter, r *http.Request) {
	var newEquipType ent.EquipmentType

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

	newEquipType.BusinessUnitID = buID
	newEquipType.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newEquipType); err != nil {
		return
	}

	createEquipType, err := services.NewEquipmentTypeOps().CreateEquipmentType(r.Context(), newEquipType)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createEquipType)
}

// UpdateEquipmentType updates a equipment type by ID.
func UpdateEquipmentType(w http.ResponseWriter, r *http.Request) {
	equipTypeID := chi.URLParam(r, "equipTypeID")
	if equipTypeID == "" {
		return
	}

	var equipTypeData ent.EquipmentType

	if err := tools.ParseBodyAndValidate(w, r, &equipTypeData); err != nil {
		return
	}

	equipTypeData.ID = uuid.MustParse(equipTypeID)

	equipmentType, err := services.NewEquipmentTypeOps().UpdateEquipmentType(r.Context(), equipTypeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, equipmentType)
}

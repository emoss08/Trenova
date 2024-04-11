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

// GetChargeTypes gets the charge types for an organization.
func GetChargeTypes(w http.ResponseWriter, r *http.Request) {
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

	chargeTypes, count, err := services.NewChargeTypeOps().GetChargeTypes(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  chargeTypes,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateChargeType creates a new charge type.
func CreateChargeType(w http.ResponseWriter, r *http.Request) {
	var newChargeType ent.ChargeType

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

	newChargeType.BusinessUnitID = buID
	newChargeType.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newChargeType); err != nil {
		return
	}

	createChargeType, err := services.NewChargeTypeOps().CreateChargeType(r.Context(), newChargeType)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createChargeType)
}

// UpdateChargeType updates a charge type by ID.
func UpdateChargeType(w http.ResponseWriter, r *http.Request) {
	chargeTypeID := chi.URLParam(r, "chargeTypeID")
	if chargeTypeID == "" {
		return
	}

	var chargeTypeData ent.ChargeType

	if err := tools.ParseBodyAndValidate(w, r, &chargeTypeData); err != nil {
		return
	}

	chargeTypeData.ID = uuid.MustParse(chargeTypeID)

	commodity, err := services.NewChargeTypeOps().UpdateChargeType(r.Context(), chargeTypeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, commodity)
}

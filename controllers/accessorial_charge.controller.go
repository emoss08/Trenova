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

// GetAccessorialCharge gets the accessorial charge for an organization.
func GetAccessorialCharge(w http.ResponseWriter, r *http.Request) {
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

	accessorialCharges, count, err := services.NewAccessorialChargeOps().GetAccessorialCharges(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  accessorialCharges,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateAccessorialCharge creates a new accessorial charge.
func CreateAccessorialCharge(w http.ResponseWriter, r *http.Request) {
	var newAccessorialCharge ent.AccessorialCharge

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

	newAccessorialCharge.BusinessUnitID = buID
	newAccessorialCharge.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newAccessorialCharge); err != nil {
		return
	}

	createAccessorialCharge, err := services.NewAccessorialChargeOps().
		CreateAccessorialCharge(r.Context(), newAccessorialCharge)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createAccessorialCharge)
}

// UpdateAccessorialCharge updates a accessorial charge by ID.
func UpdateAccessorialCharge(w http.ResponseWriter, r *http.Request) {
	accessorialChargeID := chi.URLParam(r, "accessorialChargeID")
	if accessorialChargeID == "" {
		return
	}

	var accessorialChargeData ent.AccessorialCharge

	if err := tools.ParseBodyAndValidate(w, r, &accessorialChargeData); err != nil {
		return
	}

	accessorialChargeData.ID = uuid.MustParse(accessorialChargeID)

	accessorialCharge, err := services.NewAccessorialChargeOps().UpdateAccessorialCharge(r.Context(), accessorialChargeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, accessorialCharge)
}

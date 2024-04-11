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

// GetFleetCodes gets the revenue codes for an organization.
func GetFleetCodes(w http.ResponseWriter, r *http.Request) {
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

	fleetCodes, count, err := services.NewFleetCodeOps().GetFleetCodes(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  fleetCodes,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateFleetCode creates a new fleet code.
func CreateFleetCode(w http.ResponseWriter, r *http.Request) {
	var newFleetCode ent.FleetCode

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

	newFleetCode.BusinessUnitID = buID
	newFleetCode.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newFleetCode); err != nil {
		return
	}

	createFleetCode, err := services.NewFleetCodeOps().CreateFleetCode(r.Context(), newFleetCode)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createFleetCode)
}

// UpdateFleetCode updates a fleet code by ID.
func UpdateFleetCode(w http.ResponseWriter, r *http.Request) {
	fleetCodeID := chi.URLParam(r, "fleetCodeID")
	if fleetCodeID == "" {
		return
	}

	var fleetCodeData ent.FleetCode

	if err := tools.ParseBodyAndValidate(w, r, &fleetCodeData); err != nil {
		return
	}

	fleetCodeData.ID = uuid.MustParse(fleetCodeID)

	fleetCode, err := services.NewFleetCodeOps().UpdateFleetCode(r.Context(), fleetCodeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, fleetCode)
}

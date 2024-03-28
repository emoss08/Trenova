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

// GetReasonCode gets the reason codes for an organization.
func GetReasonCode(w http.ResponseWriter, r *http.Request) {
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

	reasonCode, count, err := services.NewReasonCodeOps(r.Context()).GetReasonCode(limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  reasonCode,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateReasonCode creates a new reason code.
func CreateReasonCode(w http.ResponseWriter, r *http.Request) {
	var newReasonCode ent.ReasonCode

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

	newReasonCode.BusinessUnitID = buID
	newReasonCode.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newReasonCode); err != nil {
		return
	}

	createReasonCode, err := services.NewReasonCodeOps(r.Context()).CreateReasonCode(newReasonCode)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createReasonCode)
}

// UpdateReasonCode updates a reason code by ID.
func UpdateReasonCode(w http.ResponseWriter, r *http.Request) {
	reasonCodeID := chi.URLParam(r, "reasonCodeID")
	if reasonCodeID == "" {
		return
	}

	var reasonCodeData ent.ReasonCode

	if err := tools.ParseBodyAndValidate(w, r, &reasonCodeData); err != nil {
		return
	}

	reasonCodeData.ID = uuid.MustParse(reasonCodeID)

	reasonCode, err := services.NewReasonCodeOps(r.Context()).UpdateReasonCode(reasonCodeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, reasonCode)
}

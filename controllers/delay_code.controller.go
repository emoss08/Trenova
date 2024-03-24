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

// GetDelayCodes gets the delay code for an organization.
func GetDelayCodes(w http.ResponseWriter, r *http.Request) {
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

	delayCodes, count, err := services.NewDelayCodeOps(r.Context()).GetDelayCodes(limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  delayCodes,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateDelayCode creates a new delay code.
func CreateDelayCode(w http.ResponseWriter, r *http.Request) {
	var newDelayCode ent.DelayCode

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

	newDelayCode.BusinessUnitID = buID
	newDelayCode.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newDelayCode); err != nil {
		return
	}

	createDelaycode, err := services.NewDelayCodeOps(r.Context()).
		CreateDelayCode(newDelayCode)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createDelaycode)
}

// UpdateDelayCode updates a delay code by ID.
func UpdateDelayCode(w http.ResponseWriter, r *http.Request) {
	delayCodeID := chi.URLParam(r, "delayCodeID")
	if delayCodeID == "" {
		return
	}

	var delayCodeData ent.DelayCode

	if err := tools.ParseBodyAndValidate(w, r, &delayCodeData); err != nil {
		return
	}

	delayCodeData.ID = uuid.MustParse(delayCodeID)

	delayCode, err := services.NewDelayCodeOps(r.Context()).UpdateDelayCode(delayCodeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, delayCode)
}

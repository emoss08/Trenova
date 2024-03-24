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

// GetRevenueCodes gets the revenue codes for an organization.
func GetRevenueCodes(w http.ResponseWriter, r *http.Request) {
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

	revnueCodes, count, err := services.NewRevenueCodeOps(r.Context()).GetRevenueCodes(limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  revnueCodes,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

func CreateRevenueCode(w http.ResponseWriter, r *http.Request) {
	var newRevenueCode ent.RevenueCode

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

	newRevenueCode.BusinessUnitID = buID
	newRevenueCode.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newRevenueCode); err != nil {
		return
	}

	createRevenueCode, err := services.NewRevenueCodeOps(r.Context()).CreateRevenueCode(newRevenueCode)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createRevenueCode)
}

// UpdateRevenueCode updates a revenue code by ID.
func UpdateRevenueCode(w http.ResponseWriter, r *http.Request) {
	revenueCodeID := chi.URLParam(r, "revenueCodeID")
	if revenueCodeID == "" {
		return
	}

	var revenueCodeData ent.RevenueCode

	if err := tools.ParseBodyAndValidate(w, r, &revenueCodeData); err != nil {
		return
	}

	revenueCodeData.ID = uuid.MustParse(revenueCodeID)

	revenueCode, err := services.NewRevenueCodeOps(r.Context()).UpdateRevenueCode(revenueCodeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, revenueCode)
}

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

// GetDivisionCodes gets the division codes for an organization.
func GetDivisionCodes(w http.ResponseWriter, r *http.Request) {
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

	divisionCodes, count, err := services.NewDivisionCodeOps().GetDivisionCodes(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  divisionCodes,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateDivisionCode creates a new division code.
func CreateDivisionCode(w http.ResponseWriter, r *http.Request) {
	var newDivisionCode ent.DivisionCode

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

	newDivisionCode.BusinessUnitID = buID
	newDivisionCode.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newDivisionCode); err != nil {
		return
	}

	createDivisionCode, err := services.NewDivisionCodeOps().CreateDivisionCode(r.Context(), newDivisionCode)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createDivisionCode)
}

// UpdateDivisionCode updates a division code by ID.
func UpdateDivisionCode(w http.ResponseWriter, r *http.Request) {
	divisionCodeID := chi.URLParam(r, "divisionCodeID")
	if divisionCodeID == "" {
		return
	}

	var divisionCodeData ent.DivisionCode

	if err := tools.ParseBodyAndValidate(w, r, &divisionCodeData); err != nil {
		return
	}

	divisionCodeData.ID = uuid.MustParse(divisionCodeID)

	divisionCode, err := services.NewDivisionCodeOps().UpdateDivisionCode(r.Context(), divisionCodeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, divisionCode)
}

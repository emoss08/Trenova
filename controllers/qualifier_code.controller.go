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

// GetQualifierCodes gets the qualifier codes for an organization.
func GetQualifierCodes(w http.ResponseWriter, r *http.Request) {
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

	qualifierCodes, count, err := services.NewQualifierCodeOps().GetQualifierCodes(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  qualifierCodes,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateQualifierCode creates a new qualifier code.
func CreateQualifierCode(w http.ResponseWriter, r *http.Request) {
	var newQualifierCode ent.QualifierCode

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

	newQualifierCode.BusinessUnitID = buID
	newQualifierCode.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newQualifierCode); err != nil {
		return
	}

	createQualifierCode, err := services.NewQualifierCodeOps().CreateQualifierCode(r.Context(), newQualifierCode)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createQualifierCode)
}

// UpdateQualifierCode updates a qualifier code by ID.
func UpdateQualifierCode(w http.ResponseWriter, r *http.Request) {
	qualifierCodeID := chi.URLParam(r, "qualifierCodeID")
	if qualifierCodeID == "" {
		return
	}

	var qualifierCodeData ent.QualifierCode

	if err := tools.ParseBodyAndValidate(w, r, &qualifierCodeData); err != nil {
		return
	}

	qualifierCodeData.ID = uuid.MustParse(qualifierCodeID)

	qualifierCode, err := services.NewQualifierCodeOps().UpdateQualifierCode(r.Context(), qualifierCodeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, qualifierCode)
}

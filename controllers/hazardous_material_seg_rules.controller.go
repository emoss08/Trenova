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

// GetHazardousMaterialSegregations gets the hazardous material segregations for an organization.
func GetHazmatSegRules(w http.ResponseWriter, r *http.Request) {
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

	hazmatSegRules, count, err := services.NewHazardousMaterialSegregationOps().
		GetHazmatSegRules(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  hazmatSegRules,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateHazardousSegRule creates a new hazardous material segregation rule.
func CreateHazmatSegRule(w http.ResponseWriter, r *http.Request) {
	var newHazmatSegRule ent.HazardousMaterialSegregation

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

	newHazmatSegRule.BusinessUnitID = buID
	newHazmatSegRule.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newHazmatSegRule); err != nil {
		return
	}

	newRecord, err := services.NewHazardousMaterialSegregationOps().CreateHazmatSegRule(r.Context(), newHazmatSegRule)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, newRecord)
}

// UpdateHazardousSegRule updates a hazardous seg. rule by ID.
func UpdateHazmatSegRule(w http.ResponseWriter, r *http.Request) {
	hazmatSegRuleID := chi.URLParam(r, "hazmatSegRuleID")
	if hazmatSegRuleID == "" {
		return
	}

	var hazmatSegRuleData ent.HazardousMaterialSegregation

	if err := tools.ParseBodyAndValidate(w, r, &hazmatSegRuleData); err != nil {
		return
	}

	hazmatSegRuleData.ID = uuid.MustParse(hazmatSegRuleID)

	hazmatSegRule, err := services.NewHazardousMaterialSegregationOps().UpdateHazmatSegRule(r.Context(), hazmatSegRuleData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, hazmatSegRule)
}

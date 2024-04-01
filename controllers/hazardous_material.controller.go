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

// GetHazardousMaterial gets the hazardous material for an organization.
func GetHazardousMaterial(w http.ResponseWriter, r *http.Request) {
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

	hazardousMaterials, count, err := services.NewHazardousMaterialOps().GetHazardousMaterials(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  hazardousMaterials,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateHazardousMaterial creates a new hazardous material.
func CreateHazardousMaterial(w http.ResponseWriter, r *http.Request) {
	var newHazardousMaterial ent.HazardousMaterial

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

	newHazardousMaterial.BusinessUnitID = buID
	newHazardousMaterial.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newHazardousMaterial); err != nil {
		return
	}

	createdMaterial, err := services.NewHazardousMaterialOps().CreateHazardousMaterial(r.Context(), newHazardousMaterial)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createdMaterial)
}

// UpdateHazardousMaterial updates a hazardous material by ID.
func UpdateHazardousMaterial(w http.ResponseWriter, r *http.Request) {
	hazmatID := chi.URLParam(r, "hazmatID")
	if hazmatID == "" {
		return
	}

	var hazmatData ent.HazardousMaterial

	if err := tools.ParseBodyAndValidate(w, r, &hazmatData); err != nil {
		return
	}

	hazmatData.ID = uuid.MustParse(hazmatID)

	hazardousMaterial, err := services.NewHazardousMaterialOps().UpdateHazardousMaterial(r.Context(), hazmatData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, hazardousMaterial)
}

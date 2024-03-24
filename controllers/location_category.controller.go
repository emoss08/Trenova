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

// GetLocationCategories gets the location categories for an organization.
func GetLocationCategories(w http.ResponseWriter, r *http.Request) {
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
					Code: "internalError", Detail: "Organization ID or Business Unit ID not found in the request context",
					Attr: "orgID, buID",
				},
			},
		})

		return
	}

	locationCategories, count, err := services.NewLocationCategoryOps(r.Context()).
		GetLocationCategories(limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  locationCategories,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateLocationCategory creates a new location categories.
func CreateLocationCategory(w http.ResponseWriter, r *http.Request) {
	var newLocationCategory ent.LocationCategory

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

	newLocationCategory.BusinessUnitID = buID
	newLocationCategory.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newLocationCategory); err != nil {
		return
	}

	createLocationCategory, err := services.NewLocationCategoryOps(r.Context()).
		CreateLocationCategory(newLocationCategory)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createLocationCategory)
}

// UpdateLocationCategory updates a location categories by ID.
func UpdateLocationCategory(w http.ResponseWriter, r *http.Request) {
	locationCategoryID := chi.URLParam(r, "locationCategoryID")
	if locationCategoryID == "" {
		return
	}

	var locationCategoryData ent.LocationCategory

	if err := tools.ParseBodyAndValidate(w, r, &locationCategoryData); err != nil {
		return
	}

	locationCategoryData.ID = uuid.MustParse(locationCategoryID)

	locationCategory, err := services.NewLocationCategoryOps(r.Context()).
		UpdateLocationCategory(locationCategoryData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, locationCategory)
}

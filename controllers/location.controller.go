package controllers

import (
	"net/http"
	"time"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/location"
	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type LocationResponse struct {
	ID                 uuid.UUID             `json:"id,omitempty"`
	BusinessUnitID     uuid.UUID             `json:"businessUnitId"`
	OrganizationID     uuid.UUID             `json:"organizationId"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
	Version            int                   `json:"version" validate:"omitempty"`
	Status             location.Status       `json:"status" validate:"required,oneof=A I"`
	Code               string                `json:"code" validate:"required,max=10"`
	LocationCategoryID *uuid.UUID            `json:"locationCategoryId" validate:"omitempty"`
	Name               string                `json:"name" validate:"required"`
	Description        string                `json:"description" validate:"omitempty"`
	AddressLine1       string                `json:"addressLine1" validate:"required,max=150"`
	AddressLine2       string                `json:"addressLine2" validate:"omitempty,max=150"`
	City               string                `json:"city" validate:"required,max=150"`
	StateID            uuid.UUID             `json:"stateId" validate:"omitempty,uuid"`
	PostalCode         string                `json:"postalCode" validate:"required,max=10"`
	Longitude          float64               `json:"longitude" validate:"omitempty"`
	Latitude           float64               `json:"latitude" validate:"omitempty"`
	PlaceID            string                `json:"placeId" validate:"omitempty,max=255"`
	IsGeocoded         bool                  `json:"isGeocoded"`
	Comments           []ent.LocationComment `json:"comments"`
	Edges              ent.LocationEdges     `json:"edges"`
}

// GetLocations gets the locations for an organization.
func GetLocations(w http.ResponseWriter, r *http.Request) {
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

	locations, count, err := services.NewLocationOps().GetLocations(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	responses := make([]LocationResponse, len(locations))
	for i, location := range locations {
		// Directly assign the comments from the location object
		comments := make([]ent.LocationComment, len(location.Edges.Comments))
		for j, comment := range location.Edges.Comments {
			comments[j] = *comment
		}

		// Response for the location
		responses[i] = LocationResponse{
			ID:                 location.ID,
			BusinessUnitID:     location.BusinessUnitID,
			OrganizationID:     location.OrganizationID,
			CreatedAt:          location.CreatedAt,
			UpdatedAt:          location.UpdatedAt,
			Version:            location.Version,
			Status:             location.Status,
			Code:               location.Code,
			LocationCategoryID: location.LocationCategoryID,
			Name:               location.Name,
			Description:        location.Description,
			AddressLine1:       location.AddressLine1,
			AddressLine2:       location.AddressLine2,
			City:               location.City,
			StateID:            location.StateID,
			PostalCode:         location.PostalCode,
			Longitude:          location.Longitude,
			Latitude:           location.Latitude,
			PlaceID:            location.PlaceID,
			IsGeocoded:         location.IsGeocoded,
			Comments:           comments,
			Edges:              location.Edges,
		}
	}

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  responses,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateLocation creates a new location.
func CreateLocation(w http.ResponseWriter, r *http.Request) {
	var newLocation services.LocationRequest

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

	newLocation.BusinessUnitID = buID
	newLocation.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newLocation); err != nil {
		return
	}

	newRecord, err := services.NewLocationOps().CreateLocation(r.Context(), newLocation)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, newRecord)
}

// UpdateLocation updates a location by ID.
func UpdateLocation(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "locationID")
	if locationID == "" {
		return
	}

	var locationData services.LocationUpdateRequest

	if err := tools.ParseBodyAndValidate(w, r, &locationData); err != nil {
		return
	}

	locationData.ID = uuid.MustParse(locationID)

	location, err := services.NewLocationOps().UpdateLocation(r.Context(), locationData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, location)
}

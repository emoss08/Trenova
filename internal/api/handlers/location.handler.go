package handlers

import (
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/location"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
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
	Contacts           []ent.LocationContact `json:"contacts"`
	Edges              ent.LocationEdges     `json:"edges"`
}

// GetLocations is a handler that returns a list of locations.
//
// GET /locations
func GetLocations(s *api.Server) fiber.Handler {
	return func(c *fiber.Ctx) error {
		offset, limit, err := util.PaginationParams(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "offset, limit",
					},
				},
			})
		}

		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !buOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID or Business Unit ID not found in the request context",
						Attr:   "orgID, buID",
					},
				},
			})
		}

		entities, count, err := services.NewLocationService(s).
			GetLocations(c.UserContext(), limit, offset, orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		nextURL := util.GetNextPageURL(c, limit, offset, count)
		prevURL := util.GetPrevPageURL(c, limit, offset)

		responses := make([]LocationResponse, len(entities))
		for i, location := range entities {
			// Directly assign the comments from the location object
			comments := make([]ent.LocationComment, len(location.Edges.Comments))
			for j, comment := range location.Edges.Comments {
				comments[j] = *comment
			}

			// Directly assign the comments form the location objects
			contacts := make([]ent.LocationContact, len(location.Edges.Contacts))
			for k, contact := range location.Edges.Contacts {
				contacts[k] = *contact
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
				Contacts:           contacts,
				Edges:              location.Edges,
			}
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results:  responses,
			Count:    count,
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

// CreateLocation is a handler that creates a new location.
//
// POST /locations
func CreateLocation(s *api.Server) fiber.Handler {
	return func(c *fiber.Ctx) error {
		newEntity := new(services.LocationRequest)

		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !buOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID or Business Unit ID not found in the request context",
						Attr:   "orgID, buID",
					},
				},
			})
		}

		newEntity.BusinessUnitID = buID
		newEntity.OrganizationID = orgID

		if err := util.ParseBodyAndValidate(c, newEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "body",
					},
				},
			})
		}

		entity, err := services.NewLocationService(s).
			CreateLocation(c.UserContext(), newEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

// UpdateLocation is a handler that updates an location.
//
// PUT /locations/:locationID
func UpdateLocation(s *api.Server) fiber.Handler {
	return func(c *fiber.Ctx) error {
		locationID := c.Params("locationID")
		if locationID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "location ID is required",
						Attr:   "locationID",
					},
				},
			})
		}

		updatedEntity := new(services.LocationUpdateRequest)

		if err := util.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "request body",
					},
				},
			})
		}

		updatedEntity.ID = uuid.MustParse(locationID)

		entity, err := services.NewLocationService(s).
			UpdateLocation(c.UserContext(), updatedEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

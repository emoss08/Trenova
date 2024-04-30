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

type LocationHandler struct {
	Server            *api.Server
	Service           *services.LocationService
	PermissionService *services.PermissionService
}

func NewLocationHandler(s *api.Server) *LocationHandler {
	return &LocationHandler{
		Server:            s,
		Service:           services.NewLocationService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the location routes to the fiber app.
func (h *LocationHandler) RegisterRoutes(r fiber.Router) {
	locations := r.Group("/locations")
	locations.Get("/", h.GetLocations())
	locations.Post("/", h.CreateLocation())
	locations.Put("/:locationID", h.UpdateLocation())
}

// GetLocations is a handler that returns a list of locations.
//
// GET /locations
func (h *LocationHandler) GetLocations() fiber.Handler {
	return func(c *fiber.Ctx) error {
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

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "location.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

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

		entities, count, err := h.Service.GetLocations(c.UserContext(), limit, offset, orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		nextURL := util.GetNextPageURL(c, limit, offset, count)
		prevURL := util.GetPrevPageURL(c, limit, offset)

		responses := make([]LocationResponse, len(entities))
		for i, loc := range entities {
			// Directly assign the comments from the location object
			comments := make([]ent.LocationComment, len(loc.Edges.Comments))
			for j, comment := range loc.Edges.Comments {
				comments[j] = *comment
			}

			// Directly assign the comments form the location objects
			contacts := make([]ent.LocationContact, len(loc.Edges.Contacts))
			for k, contact := range loc.Edges.Contacts {
				contacts[k] = *contact
			}

			// Response for the location
			responses[i] = LocationResponse{
				ID:                 loc.ID,
				BusinessUnitID:     loc.BusinessUnitID,
				OrganizationID:     loc.OrganizationID,
				CreatedAt:          loc.CreatedAt,
				UpdatedAt:          loc.UpdatedAt,
				Version:            loc.Version,
				Status:             loc.Status,
				Code:               loc.Code,
				LocationCategoryID: loc.LocationCategoryID,
				Name:               loc.Name,
				Description:        loc.Description,
				AddressLine1:       loc.AddressLine1,
				AddressLine2:       loc.AddressLine2,
				City:               loc.City,
				StateID:            loc.StateID,
				PostalCode:         loc.PostalCode,
				Longitude:          loc.Longitude,
				Latitude:           loc.Latitude,
				PlaceID:            loc.PlaceID,
				IsGeocoded:         loc.IsGeocoded,
				Comments:           comments,
				Contacts:           contacts,
				Edges:              loc.Edges,
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
func (h *LocationHandler) CreateLocation() fiber.Handler {
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

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "location.add")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		newEntity.BusinessUnitID = buID
		newEntity.OrganizationID = orgID

		if err := util.ParseBodyAndValidate(c, newEntity); err != nil {
			return err
		}

		entity, err := h.Service.CreateLocation(c.UserContext(), newEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

// UpdateLocation is a handler that updates a location.
//
// PUT /locations/:locationID
func (h *LocationHandler) UpdateLocation() fiber.Handler {
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

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "location.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		updatedEntity := new(services.LocationUpdateRequest)

		if err := util.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return err
		}

		updatedEntity.ID = uuid.MustParse(locationID)

		entity, err := h.Service.UpdateLocation(c.UserContext(), updatedEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

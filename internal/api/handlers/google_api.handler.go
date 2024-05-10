package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type GoogleAPIHandler struct {
	Server            *api.Server
	Service           *services.GoogleAPIService
	PermissionService *services.PermissionService
}

func NewGoogleAPIHandler(s *api.Server) *GoogleAPIHandler {
	return &GoogleAPIHandler{
		Server:            s,
		Service:           services.NewGoogleAPIService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the GoogleAPIHandler.
func (h *GoogleAPIHandler) RegisterRoutes(r fiber.Router) {
	googleAPI := r.Group("/google-api")
	googleAPI.Get("/", h.GetGoogleAPI())
	googleAPI.Put("/:googleAPIID", h.UpdateGoogleAPI())
}

// GetGoogleAPI is a handler that returns the Google api settings for an organization.
//
// GET /google-api
func (h *GoogleAPIHandler) GetGoogleAPI() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "googleapi.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		entity, err := h.Service.GetGoogleAPI(c.UserContext(), orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h *GoogleAPIHandler) UpdateGoogleAPI() fiber.Handler {
	return func(c *fiber.Ctx) error {
		googleAPIID := c.Params("googleAPIID")
		if googleAPIID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Billing Control ID is required",
						Attr:   "googleAPIID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "googleapi.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		data := new(ent.GoogleApi)
		if err := util.ParseBodyAndValidate(c, data); err != nil {
			return err
		}

		data.ID = uuid.MustParse(googleAPIID)

		updatedEntity, err := h.Service.UpdateGoogleAPI(c.UserContext(), data)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedEntity)
	}
}

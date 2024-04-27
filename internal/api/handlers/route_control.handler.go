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

type RouteControlHandler struct {
	Server            *api.Server
	Service           *services.RouteControlService
	PermissionService *services.PermissionService
}

func NewRouteControlHandler(s *api.Server) *RouteControlHandler {
	return &RouteControlHandler{
		Server:            s,
		Service:           services.NewRouteControlService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// GetRouteControl is a handler that returns the route control for an organization.
//
// GET /route-control
func (h *RouteControlHandler) GetRouteControl() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "routecontrol.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		entity, err := h.Service.GetRouteControl(c.UserContext(), orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

// UpdateRouteControlByID is a handler that updates the route control for an organization.
//
// PUT /route-control/:routeControlID
func (h *RouteControlHandler) UpdateRouteControlByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		routeControlID := c.Params("routeControlID")
		if routeControlID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Route Control ID is required",
						Attr:   "routeControlID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "routecontrol.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		data := new(ent.RouteControl)

		if err := util.ParseBodyAndValidate(c, data); err != nil {
			return err
		}

		data.ID = uuid.MustParse(routeControlID)

		updatedEntity, err := h.Service.UpdateRouteControl(c.UserContext(), data)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedEntity)
	}
}

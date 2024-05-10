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

type FeasibilityToolControlHandler struct {
	Server            *api.Server
	Service           *services.FeasibilityToolControlService
	PermissionService *services.PermissionService
}

func NewFeasibilityToolControlHandler(s *api.Server) *FeasibilityToolControlHandler {
	return &FeasibilityToolControlHandler{
		Server:            s,
		Service:           services.NewFeasibilityToolControlService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the FeasibilityToolControlHandler.
func (h *FeasibilityToolControlHandler) RegisterRoutes(r fiber.Router) {
	feasibilityToolControlAPI := r.Group("/feasibility-tool-control")
	feasibilityToolControlAPI.Get("/", h.GetFeasibilityToolControl())
	feasibilityToolControlAPI.Put("/:feasibilityToolControlID", h.UpdateFeasibilityToolControl())
}

// GetFeasibilityToolControl is a handler that returns the feasibility tool control for an organization.
//
// GET /billing-control
func (h *FeasibilityToolControlHandler) GetFeasibilityToolControl() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "feasibilitytoolcontrol.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		entity, err := h.Service.GetFeasibilityToolControl(c.UserContext(), orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h *FeasibilityToolControlHandler) UpdateFeasibilityToolControl() fiber.Handler {
	return func(c *fiber.Ctx) error {
		feasibilityToolControlID := c.Params("feasibilityToolControlID")
		if feasibilityToolControlID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Billing Control ID is required",
						Attr:   "feasibilityToolControlID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "feasibilitytoolcontrol.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		data := new(ent.FeasibilityToolControl)

		if err := util.ParseBodyAndValidate(c, data); err != nil {
			return err
		}

		data.ID = uuid.MustParse(feasibilityToolControlID)

		updatedEntity, err := h.Service.UpdateFeasibilityToolControl(c.UserContext(), data)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedEntity)
	}
}

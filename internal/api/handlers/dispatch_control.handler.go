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

type DispatchControlHandler struct {
	Service           *services.DispatchControlService
	PermissionService *services.PermissionService
}

func NewDispatchControlHandler(s *api.Server) *DispatchControlHandler {
	return &DispatchControlHandler{
		Service:           services.NewDispatchControlService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the DispatchControlHandler.
func (h *DispatchControlHandler) RegisterRoutes(r fiber.Router) {
	dispatchControlAPI := r.Group("/dispatch-control")
	dispatchControlAPI.Get("/", h.GetDispatchControl())
	dispatchControlAPI.Put("/:dispatchControlID", h.UpdateDispatchControlByID())
}

// GetDispatchControl is a handler that returns the dispatch control for an organization.
//
// GET /dispatch-control
func (h *DispatchControlHandler) GetDispatchControl() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "dispatchcontrol.add")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		entity, err := h.Service.GetDispatchControl(c.UserContext(), orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

// UpdateDispatchControlByID is a handler that updates the dispatch control for an organization.
//
// PUT /dispatch-control/:dispatchControlID
func (h *DispatchControlHandler) UpdateDispatchControlByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		dispatchControlID := c.Params("dispatchControlID")
		if dispatchControlID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Dispatch Control ID is required",
						Attr:   "dispatchControlID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "dispatchcontrol.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		data := new(ent.DispatchControl)

		if err := util.ParseBodyAndValidate(c, data); err != nil {
			return err
		}

		data.ID = uuid.MustParse(dispatchControlID)

		updatedEntity, err := h.Service.UpdateDispatchControl(c.UserContext(), data)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedEntity)
	}
}

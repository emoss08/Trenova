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

type EmailControlHandler struct {
	Service           *services.EmailControlService
	PermissionService *services.PermissionService
}

func NewEmailControlHandler(s *api.Server) *EmailControlHandler {
	return &EmailControlHandler{
		Service:           services.NewEmailControlService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the EmailControlHandler.
func (h *EmailControlHandler) RegisterRoutes(r fiber.Router) {
	emailControlAPI := r.Group("/email-control")
	emailControlAPI.Get("/", h.GetEmailControl())
	emailControlAPI.Put("/:emailControlID", h.UpdateEmailControl())
}

// GetEmailControl is a handler that returns the billing control for an organization.
//
// GET /billing-control
func (h *EmailControlHandler) GetEmailControl() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "emailcontrol.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		entity, err := h.Service.GetEmailControl(c.UserContext(), orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h *EmailControlHandler) UpdateEmailControl() fiber.Handler {
	return func(c *fiber.Ctx) error {
		emailControlID := c.Params("emailControlID")
		if emailControlID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Billing Control ID is required",
						Attr:   "emailControlID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "emailcontrol.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		data := new(ent.EmailControl)

		if err := util.ParseBodyAndValidate(c, data); err != nil {
			return err
		}

		data.ID = uuid.MustParse(emailControlID)

		updatedEntity, err := h.Service.UpdateEmailControl(c.UserContext(), data)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedEntity)
	}
}

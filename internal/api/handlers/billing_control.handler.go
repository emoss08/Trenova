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

type BillingControlHandler struct {
	Server            *api.Server
	Service           *services.BillingControlService
	PermissionService *services.PermissionService
}

func NewBillingControlHandler(s *api.Server) *BillingControlHandler {
	return &BillingControlHandler{
		Server:            s,
		Service:           services.NewBillingControlService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the BillingControlHandler.
func (h *BillingControlHandler) RegisterRoutes(r fiber.Router) {
	billingControlAPI := r.Group("/billing-control")
	billingControlAPI.Get("/", h.GetBillingControl())
	billingControlAPI.Put("/:billingControlID", h.UpdateBillingControl())
}

// GetBillingControl is a handler that returns the billing control for an organization.
//
// GET /billing-control
func (h *BillingControlHandler) GetBillingControl() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "billingcontrol.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		entity, err := h.Service.GetBillingControl(c.UserContext(), orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h *BillingControlHandler) UpdateBillingControl() fiber.Handler {
	return func(c *fiber.Ctx) error {
		billingControlID := c.Params("billingControlID")
		if billingControlID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Billing Control ID is required",
						Attr:   "billingControlID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "billingcontrol.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		data := new(ent.BillingControl)

		if err := util.ParseBodyAndValidate(c, data); err != nil {
			return err
		}

		data.ID = uuid.MustParse(billingControlID)

		updatedEntity, err := h.Service.UpdateBillingControl(c.UserContext(), data)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedEntity)
	}
}

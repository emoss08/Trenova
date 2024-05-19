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

type AccountingControlHandler struct {
	Service           *services.AccountingControlService
	PermissionService *services.PermissionService
}

// NewAccountingControlHandler returns a new AccountingControlHandler.
func NewAccountingControlHandler(s *api.Server) *AccountingControlHandler {
	return &AccountingControlHandler{
		Service:           services.NewAccountingControlService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the AccountingControlHandler.
func (h *AccountingControlHandler) RegisterRoutes(r fiber.Router) {
	accountingControlAPI := r.Group("/accounting-control")
	accountingControlAPI.Get("/", h.GetAccountingControl())
	accountingControlAPI.Put("/:accountingControlID", h.UpdateAccountingControlByID())
}

// GetAccountingControl is a handler that returns the accounting control for an organization.
//
// GET /accounting-control
func (h *AccountingControlHandler) GetAccountingControl() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "accountingcontrol.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		entity, err := h.Service.
			GetAccountingControl(c.UserContext(), orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

// UpdateAccountingControlByID is a handler that updates the accounting control for an organization.
//
// PUT /accounting-control/:accountingControlID
func (h *AccountingControlHandler) UpdateAccountingControlByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountingControlID := c.Params("accountingControlID")
		if accountingControlID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Accounting Control ID is required",
						Attr:   "accountingControlID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "accountingcontrol.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		data := new(ent.AccountingControl)

		if err := util.ParseBodyAndValidate(c, data); err != nil {
			return err
		}

		data.ID = uuid.MustParse(accountingControlID)

		updatedEntity, err := h.Service.UpdateAccountingControl(c.UserContext(), data)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedEntity)
	}
}

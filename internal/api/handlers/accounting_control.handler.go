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
	Server  *api.Server
	Service *services.AccountingControlService
}

// NewAccountingControlHandler returns a new AccountingControlHandler.
func NewAccountingControlHandler(s *api.Server) *AccountingControlHandler {
	return &AccountingControlHandler{
		Server:  s,
		Service: services.NewAccountingControlService(s),
	}
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

		entity, err := h.Service.GetAccountingControl(c.UserContext(), orgID, buID)
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

		data := new(ent.AccountingControl)

		if err := util.ParseBodyAndValidate(c, data); err != nil {
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

		data.ID = uuid.MustParse(accountingControlID)

		updatedEntity, err := h.Service.UpdateAccountingControl(c.UserContext(), data)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedEntity)
	}
}

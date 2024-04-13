package handlers

import (
	"log"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetAccountingControl is a handler that returns the accounting control for an organization.
func GetAccountingControl(s *api.Server) fiber.Handler {
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

		entity, err := services.NewAccountingControlService(s).GetAccountingControl(c.UserContext(), orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

// UpdateAccountingControl is a handler that updates the accounting control for an organization.
//
// PUT /accounting-control/:accountingControlID
func UpdateAccountingControlByID(s *api.Server) fiber.Handler {
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

		if err := c.BodyParser(data); err != nil {
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

		if _, err := uuid.Parse(accountingControlID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Accounting Control ID must be a valid UUID",
						Attr:   "accountingControlID",
					},
				},
			})
		}

		data.ID = uuid.MustParse(accountingControlID)

		log.Printf("Accounting Control Data: %+v", data)

		updatedAccountingControl, err := s.Client.AccountingControl.UpdateOneID(data.ID).
			SetRecThreshold(data.RecThreshold).
			SetRecThresholdAction(data.RecThresholdAction).
			SetAutoCreateJournalEntries(data.AutoCreateJournalEntries).
			SetJournalEntryCriteria(data.JournalEntryCriteria).
			SetRestrictManualJournalEntries(data.RestrictManualJournalEntries).
			SetRequireJournalEntryApproval(data.RequireJournalEntryApproval).
			SetEnableRecNotifications(data.EnableRecNotifications).
			SetHaltOnPendingRec(data.HaltOnPendingRec).
			SetNillableCriticalProcesses(data.CriticalProcesses).
			SetNillableDefaultRevAccountID(data.DefaultRevAccountID).
			SetNillableDefaultExpAccountID(data.DefaultExpAccountID).
			Save(c.UserContext())
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedAccountingControl)
	}
}

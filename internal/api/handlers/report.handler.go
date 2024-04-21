package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ReportHandler struct {
	Server  *api.Server
	Service *services.ReportService
}

func NewReportHandler(s *api.Server) *ReportHandler {
	return &ReportHandler{
		Server:  s,
		Service: services.NewReportService(s),
	}
}

// GetColumnNames returns the column names for a given table name.
func (h *ReportHandler) GetColumnNames() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableName := c.Query("tableName")
		if tableName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "query parameter 'tableName' is required",
						Attr:   "tableName",
					},
				},
			})
		}

		columns, count, err := h.Service.GetColumnsByTableName(c.UserContext(), tableName)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results: columns,
			Count:   count,
		})
	}
}

// GenerateReport generates a report based on the request.
func (h *ReportHandler) GenerateReport() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request services.GenerateReportRequest

		if err := c.BodyParser(&request); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "request body is invalid",
					},
				},
			})
		}

		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)
		userID, userOK := c.Locals(util.CTXUserID).(uuid.UUID)

		if !ok || !buOK || !userOK {
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

		entity, err := h.Service.
			GenerateReport(c.UserContext(), request, userID, orgID, buID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: err.Error(),
					},
				},
			})
		}

		// Generate a notification for the user.
		_, err = services.NewUserNotificationService(h.Server).CreateUserNotification(
			c.UserContext(), orgID, buID, userID, "New Report is available", "Sucessfully Generated Report. Click here to download", entity.ReportURL,
		)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

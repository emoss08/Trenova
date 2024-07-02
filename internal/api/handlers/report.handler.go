package handlers

import (
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type ReportHandler struct {
	logger              *zerolog.Logger
	service             *services.ReportService
	notificationService *services.UserNotificationService
	permissionService   *services.PermissionService
}

func NewReportHandler(s *server.Server) *ReportHandler {
	return &ReportHandler{
		logger:              s.Logger,
		service:             services.NewReportService(s),
		notificationService: services.NewUserNotificationService(s),
		permissionService:   services.NewPermissionService(s),
	}
}

func (h *ReportHandler) RegisterRoutes(r fiber.Router) {
	reportAPI := r.Group("/reports")
	reportAPI.Get("/column-names", h.getColumnNames())
	reportAPI.Post("/generate", h.generateReport())
}

func (h *ReportHandler) getColumnNames() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableName := c.Query("tableName")
		if tableName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Table Name is required",
			})
		}
		columns, relationships, count, err := h.service.GetColumnsByTableName(c.UserContext(), tableName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get table names",
			})
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[any]{
			Results: map[string]any{
				"columns":       columns,
				"relationships": relationships,
			},
			Count: count,
		})
	}
}

func (h *ReportHandler) generateReport() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request services.GenerateReportRequest

		if err := utils.ParseBodyAndValidate(c, &request); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)
		userID, userOK := c.Locals(utils.CTXUserID).(uuid.UUID)

		if !ok || !orgOK || !userOK {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization, Business Unit ID, User ID not found in context",
			})
		}

		request.BusinessUnitID = buID
		request.OrganizationID = orgID
		request.UserID = userID

		entity, err := h.service.GenerateReport(c.UserContext(), request, userID, orgID, buID)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to generate report")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to generate report. Don't worry, we're working on it!",
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

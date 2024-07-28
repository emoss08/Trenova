// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package handlers

import (
	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type ReportHandler struct {
	logger              *config.ServerLogger
	service             *services.ReportService
	notificationService *services.UserNotificationService
}

func NewReportHandler(s *server.Server) *ReportHandler {
	return &ReportHandler{
		logger:              s.Logger,
		service:             services.NewReportService(s),
		notificationService: services.NewUserNotificationService(s),
	}
}

func (h ReportHandler) RegisterRoutes(r fiber.Router) {
	reportAPI := r.Group("/reports")
	reportAPI.Get("/column-names", h.getColumnNames())
	reportAPI.Post("/generate", h.generateReport())
}

func (h ReportHandler) getColumnNames() fiber.Handler {
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

// TODO(WOLFRED): Add audit log service here
func (h ReportHandler) generateReport() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		var request services.GenerateReportRequest

		if err = utils.ParseBodyAndValidate(c, &request); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		request.BusinessUnitID = ids.BusinessUnitID
		request.OrganizationID = ids.OrganizationID
		request.UserID = ids.UserID

		entity, err := h.service.GenerateReport(c.UserContext(), request, ids.UserID, ids.OrganizationID, ids.BusinessUnitID)
		if err != nil {
			h.logger.Error().Str("userID", ids.UserID.String()).Err(err).Msg("Failed to generate report.")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to generate report. Don't worry, we're working on it!",
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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

func (h ReportHandler) generateReport() fiber.Handler {
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

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
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type UserTaskHandler struct {
	logger              *zerolog.Logger
	service             *services.UserTaskService
	notificationService *services.UserNotificationService
	websocketService    *services.WebsocketService
}

func NewUserTaskHandler(s *server.Server) *UserTaskHandler {
	return &UserTaskHandler{
		logger:              s.Logger,
		service:             services.NewUserTaskService(s),
		notificationService: services.NewUserNotificationService(s),
		websocketService:    services.NewWebsocketService(s),
	}
}

func (h UserTaskHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/user-tasks")
	api.Get("/", h.getUserTasks())
}

// UpdateTaskStatus updates the status of a task.
//
// This function is called by the python microservice to update the status of a task.
// The python microservice sends a POST request to the /user-tasks/update endpoint.
func (h UserTaskHandler) UpdateTaskStatus(c *fiber.Ctx) error {
	update := new(services.TaskStatusUpdate)
	if err := c.BodyParser(update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	message := services.Message{
		Type:     update.Status,
		Title:    "Task Completed Successfully",
		Content:  "Report job completed successfully. Check your inbox for the requested report",
		ClientID: update.ClientID,
	}

	h.websocketService.NotifyClient(update.ClientID, message)

	organizationID := uuid.MustParse(update.OrganizationID)
	businessUnitID := uuid.MustParse(update.BusinessUnitID)
	userID := uuid.MustParse(update.ClientID)

	// Create a user notification.
	if err := h.notificationService.CreateUserNotification(c.UserContext(),
		organizationID,
		businessUnitID,
		userID,
		"Report job completed successfully", "Report job completed successfully. Click here to download.", update.Result); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
			Code:    fiber.StatusInternalServerError,
			Message: "Failed to create user notification",
		})
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h UserTaskHandler) getUserTasks() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)
		userID, userOK := c.Locals(utils.CTXUserID).(uuid.UUID)

		if !ok || !orgOK || !userOK {
			h.logger.Error().Msg("UserTaskHandler: Organization, Business Unit ID or User ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization, Business Unit ID or User ID not found in context",
			})
		}

		entities, cnt, err := h.service.GetTasksByUserID(c.Context(), userID, buID, orgID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get user tasks",
			})
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.UserTask]{
			Results: entities,
			Count:   cnt,
		})
	}
}

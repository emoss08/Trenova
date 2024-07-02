package handlers

import (
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type UserNotificationHandler struct {
	logger  *zerolog.Logger
	service *services.UserNotificationService
}

func NewUserNotificationHandler(s *server.Server) *UserNotificationHandler {
	return &UserNotificationHandler{
		logger:  s.Logger,
		service: services.NewUserNotificationService(s),
	}
}

func (unh *UserNotificationHandler) RegisterRoutes(r fiber.Router) {
	unAPI := r.Group("/user-notifications")
	unAPI.Get("/", unh.getUserNotifications())
}

type UserNotificationResponse struct {
	UnreadCount int                        `json:"unreadCount"`
	UnreadList  []*models.UserNotification `json:"unreadList"`
}

func (unh *UserNotificationHandler) getUserNotifications() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)
		userID, userOK := c.Locals(utils.CTXUserID).(uuid.UUID)

		if !ok || !orgOK || !userOK {
			unh.logger.Error().Msg("UserNotificationHandler: Organization, Business Unit, or User ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization, Business Unit, or User ID not found in context",
			})
		}

		if c.Query("markAsRead") == "true" {
			if err := unh.service.MarkNotificationsAsRead(c.UserContext(), orgID, buID, userID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
					Code:    fiber.StatusInternalServerError,
					Message: "Failed to mark notifications as read",
				})
			}
		}

		amount := utils.StringToInt(c.Query("amount"), 10)

		entities, cnt, err := unh.service.GetUserNotifications(c.UserContext(), amount, userID, buID, orgID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get user notifications",
			})
		}

		return c.Status(fiber.StatusOK).JSON(UserNotificationResponse{
			UnreadCount: cnt,
			UnreadList:  entities,
		})
	}
}

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

type UserNotificationHandler struct {
	Server  *api.Server
	Service *services.UserNotificationService
}

func NewUserNotificationHandler(s *api.Server) *UserNotificationHandler {
	return &UserNotificationHandler{
		Server:  s,
		Service: services.NewUserNotificationService(s),
	}
}

type UserNotificationResponse struct {
	UnreadCount int                     `json:"unreadCount"`
	UnreadList  []*ent.UserNotification `json:"unreadList"`
}

func (h *UserNotificationHandler) GetUserNotifications() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)
		userID, userOK := c.Locals(util.CTXUserID).(uuid.UUID)

		if !ok || !buOK || !userOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID, Business Unit ID, or User ID not found in the request context",
						Attr:   "orgID, buID, userID",
					},
				},
			})
		}

		markAsRead := c.Query("markAsRead") == "true"
		if markAsRead {
			if err := h.Service.MarkNotificationsAsRead(c.UserContext(), orgID, buID, userID); err != nil {
				errorResponse := util.CreateDBErrorResponse(err)
				return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
			}
		}

		amount := util.ConvertToInt(c.Query("amount"), 10)

		entities, count, err := h.Service.GetUserNotifications(
			c.UserContext(), amount, userID, buID, orgID,
		)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(UserNotificationResponse{
			UnreadCount: count,
			UnreadList:  entities,
		})
	}
}

package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserHandler struct {
	Server  *api.Server
	Service *services.UserService
}

func NewUserHandler(s *api.Server) *UserHandler {
	return &UserHandler{
		Server:  s,
		Service: services.NewUserService(s),
	}
}

func (h *UserHandler) GetAuthenticatedUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals(util.CTXUserID).(uuid.UUID)

		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "User ID not found in the request context",
						Attr:   "userID",
					},
				},
			})
		}

		user, err := h.Service.GetAuthenticatedUser(c.UserContext(), userID)
		if err != nil {
			errResp := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errResp)
		}

		return c.Status(fiber.StatusOK).JSON(user)
	}
}

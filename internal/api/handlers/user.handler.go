package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GetAuthenticatedUser(s *api.Server) fiber.Handler {
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

		user, err := services.NewUserService(s).GetAuthenticatedUser(c.UserContext(), userID)
		if err != nil {
			errorRsponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorRsponse)
		}

		return c.Status(fiber.StatusOK).JSON(user)
	}
}

package middleware

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
)

// New is a middleware that sets the user, organization, and business unit IDs in the request context.
func New(s *api.Server) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := s.Session.Get(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "InternalServerError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalServerError",
						Detail: "Internal server error",
						Attr:   "session",
					},
				},
			})
		}

		userID, orgID, buID, ok := util.GetSessionDetails(sess)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(types.ValidationErrorResponse{
				Type: "Unauthorized",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "unauthorized",
						Detail: "Unauthorized",
						Attr:   "session",
					},
				},
			})
		}

		c.Locals(util.CTXUserID, userID)
		c.Locals(util.CTXOrganizationID, orgID)
		c.Locals(util.CTXBusinessUnitID, buID)

		return c.Next()
	}
}

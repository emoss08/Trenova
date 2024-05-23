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
		log := util.LogFromFiberContext(c).With().Str("middleware", "session").Logger()

		// Get session
		sess, err := s.Session.Get(c)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get session")
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "InternalServerError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalServerError",
						Detail: "Failed to retrieve session information",
						Attr:   "session",
					},
				},
			})
		}

		// Extract session details
		userID, orgID, buID, ok := util.GetSessionDetails(sess)
		if !ok {
			log.Warn().Msg("Session details not found or invalid")
			return c.Status(fiber.StatusUnauthorized).JSON(types.ValidationErrorResponse{
				Type: "Unauthorized",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "unauthorized",
						Detail: "Invalid session details",
						Attr:   "session",
					},
				},
			})
		}

		// Set context locals securely
		c.Locals(util.CTXUserID, userID)
		c.Locals(util.CTXOrganizationID, orgID)
		c.Locals(util.CTXBusinessUnitID, buID)

		return c.Next()
	}
}

package handlers

import (
	"log"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
)

// AuthenticateUser authenticates a user and sets the session values.
func AuthenticateUser(s *api.Server) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var loginRequest struct {
			Username string `json:"username" validate:"required"`
			Password string `json:"password" validate:"required"`
		}

		sess, err := s.Session.Get(c)
		if err != nil {
			log.Printf("Error getting session: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalServerError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalServerError",
						Detail: "Internal server error",
						Attr:   "session",
					},
				},
			})
		}

		if err = c.BodyParser(&loginRequest); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "badRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "badRequest",
						Detail: "Invalid request body",
						Attr:   "body",
					},
				},
			})
		}

		// Authenticate the user
		user, err := services.NewAuthenticationService(s).
			AuthenticateUser(c.UserContext(), loginRequest.Username, loginRequest.Password)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(types.ValidationErrorResponse{
				Type: "unauthorized",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "authenticationError",
						Detail: "Invalid username or password",
						Attr:   "username",
					},
					{
						Code:   "authenticationError",
						Detail: "Invalid username or password",
						Attr:   "password",
					},
				},
			})
		}

		// Set the session values
		sess.Set(util.CTXUserID, user.ID)
		sess.Set(util.CTXOrganizationID, user.OrganizationID)
		sess.Set(util.CTXBusinessUnitID, user.BusinessUnitID)

		// Set in context
		c.Locals(util.CTXUserID, user.ID)
		c.Locals(util.CTXOrganizationID, user.OrganizationID)
		c.Locals(util.CTXBusinessUnitID, user.BusinessUnitID)

		// Save the session.
		if err = sess.Save(); err != nil {
			log.Printf("Error saving session: %v", err)
			s.Logger.Panic().Msg("Failed to save session")
		}

		return c.JSON(user)
	}
}

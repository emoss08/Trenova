package handlers

import (
	"log"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
)

type AuthenticationHandler struct {
	Server  *api.Server
	Service *services.AuthenticationService
}

func NewAuthenticationHandler(s *api.Server) *AuthenticationHandler {
	return &AuthenticationHandler{
		Server:  s,
		Service: services.NewAuthenticationService(s),
	}
}

// CheckEmail checks if an email address exists in the database.
//
// POST /auth/check-email
func (h *AuthenticationHandler) CheckEmail() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var checkEmailRequest struct {
			EmailAddress string `json:"emailAddress" validate:"required,email"`
		}

		if err := util.ParseBodyAndValidate(c, &checkEmailRequest); err != nil {
			return err
		}

		// Check if the email exists
		resp, err := h.Service.
			CheckEmail(c.UserContext(), checkEmailRequest.EmailAddress)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalServerError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalServerError",
						Detail: "Internal server error",
						Attr:   "emailAddress",
					},
				},
			})
		}

		return c.Status(fiber.StatusOK).JSON(resp)
	}
}

// AuthenticateUser authenticates a user and sets the session values.
//
// POST /auth
func (h *AuthenticationHandler) AuthenticateUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var loginRequest struct {
			EmailAddress string `json:"emailAddress" validate:"required"`
			Password     string `json:"password" validate:"required"`
		}

		sess, err := h.Server.Session.Get(c)
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
		user, err := h.Service.AuthenticateUser(
			c.UserContext(), loginRequest.EmailAddress, loginRequest.Password,
		)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(types.ValidationErrorResponse{
				Type: "unauthorized",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "authenticationError",
						Detail: "Invalid email address or password",
						Attr:   "emailAddress",
					},
					{
						Code:   "authenticationError",
						Detail: "Invalid email address or password",
						Attr:   "password",
					},
				},
			})
		}

		// Set the session values
		sess.Set(string(util.CTXUserID), user.ID)
		sess.Set(string(util.CTXOrganizationID), user.OrganizationID)
		sess.Set(string(util.CTXBusinessUnitID), user.BusinessUnitID)

		// Set in context
		c.Locals(util.CTXUserID, user.ID)
		c.Locals(util.CTXOrganizationID, user.OrganizationID)
		c.Locals(util.CTXBusinessUnitID, user.BusinessUnitID)

		// Save the session.
		if err = sess.Save(); err != nil {
			log.Printf("Error saving session: %v", err)
			h.Server.Logger.Panic().Msg("Failed to save session")
		}

		return c.JSON(user)
	}
}

// LogoutUser logs out the user and clears the session.
//
// POST /auth/logout
func (h *AuthenticationHandler) LogoutUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := h.Server.Session.Get(c)
		if err != nil {
			h.Server.Logger.Error().Err(err).Msg("Error getting session")
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

		// Clear the session
		if err = sess.Destroy(); err != nil {
			h.Server.Logger.Error().Err(err).Msg("Error destroying session")
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

		return c.SendStatus(fiber.StatusNoContent)
	}
}

package handlers

import (
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

// AuthenticationHandler is a struct that handles authentication-related requests.
type AuthenticationHandler struct {
	Server  *api.Server
	Logger  *zerolog.Logger
	Service *services.AuthenticationService
}

// NewAuthenticationHandler creates a new authentication handler.
func NewAuthenticationHandler(s *api.Server) *AuthenticationHandler {
	return &AuthenticationHandler{
		Server:  s,
		Logger:  s.Logger,
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

		if err := c.BodyParser(&loginRequest); err != nil {
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

		if user.Status == "I" {
			return c.Status(fiber.StatusUnauthorized).JSON(types.ValidationErrorResponse{
				Type: "unauthorized",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "authenticationError",
						Detail: "User is no longer active. Please contact support.",
						Attr:   "emailAddress",
					},
				},
			})
		}

		claims := jwt.MapClaims{
			"userID":         user.ID,
			"organizationID": user.OrganizationID,
			"businessUnitID": user.BusinessUnitID,
			"exp":            time.Now().Add(time.Hour * 72).Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

		privateKey, _, err := middleware.LoadKeys()
		if err != nil {
			h.Logger.Error().Err(err).Msg("Error loading keys")
		}

		t, err := token.SignedString(privateKey)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Error signing token")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Internal server error",
			})
		}

		c.Cookie(&fiber.Cookie{
			Name:     "trenova-token",
			Value:    t,
			Expires:  time.Now().Add(time.Hour * 72), // 3 days
			SameSite: "Lax",
			// HTTPOnly: true,
			// Secure:   true,
		})

		return c.JSON(fiber.Map{
			"token": t,
		})
	}
}

// LogoutUser logs out the user and clears the session.
//
// POST /auth/logout
func (h *AuthenticationHandler) LogoutUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Clear the user from context
		c.Locals(util.CTXOrganizationID, nil)
		c.Locals(util.CTXBusinessUnitID, nil)
		c.Locals(util.CTXUserID, nil)

		// Expire the cookie
		c.Cookie(&fiber.Cookie{
			Name:     "trenova-token",
			Value:    "",
			Expires:  time.Now().Add(-time.Hour),
			SameSite: "Lax",
			// HTTPOnly: true,
			// Secure:   true,
		})

		return c.SendStatus(fiber.StatusNoContent)
	}
}

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
	"fmt"
	"time"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type AuthenticationHandler struct {
	logger  *config.ServerLogger
	service *services.AuthenticationService
}

func NewAuthenticationHandler(s *server.Server) *AuthenticationHandler {
	return &AuthenticationHandler{
		logger:  s.Logger,
		service: services.NewAuthenticationService(s),
	}
}

func (ah AuthenticationHandler) RegisterRoutes(r fiber.Router) {
	authAPI := r.Group("/auth")
	authAPI.Post("/check-email", ah.checkEmail())
	authAPI.Post("/login", ah.authenticateUser())
}

func (ah AuthenticationHandler) checkEmail() fiber.Handler {
	return func(c *fiber.Ctx) error {
		req := new(types.CheckEmailRequest)

		if err := utils.ParseBodyAndValidate(c, req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		resp, err := ah.service.CheckEmail(c.UserContext(), req.EmailAddress)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ProblemDetail{
				Type:     "invalid",
				Title:    "Invalid Request",
				Status:   fiber.StatusBadRequest,
				Detail:   "Email address does not exist. Please Try again.",
				Instance: fmt.Sprintf("%s/probs/validation-error", c.BaseURL()),
			})
		}

		return c.Status(fiber.StatusOK).JSON(resp)
	}
}

func (ah AuthenticationHandler) authenticateUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		req := new(types.LoginRequest)

		if err := utils.ParseBodyAndValidate(c, req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		entity, field, err := ah.service.AuthenticateUser(c.UserContext(), req.EmailAddress, req.Password)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ProblemDetail{
				Type:     "invalid",
				Title:    "Invalid Request",
				Status:   fiber.StatusBadRequest,
				Detail:   err.Error(),
				Instance: fmt.Sprintf("%s/probs/validation-error", c.BaseURL()),
				InvalidParams: []types.InvalidParam{
					{
						Name:   field,
						Reason: err.Error(),
					},
				},
			})
		}

		if entity.Status == property.StatusInactive {
			return c.Status(fiber.StatusBadRequest).JSON(types.ProblemDetail{
				Type:     "invalid",
				Title:    "Invalid Request",
				Status:   fiber.StatusBadRequest,
				Detail:   "User is inactive",
				Instance: fmt.Sprintf("%s/probs/validation-error", c.BaseURL()),
				InvalidParams: []types.InvalidParam{
					{
						Name:   "status",
						Reason: "User is inactive",
					},
				},
			})
		}

		token := jwt.New(jwt.SigningMethodRS256)

		claims := token.Claims.(jwt.MapClaims)
		claims["userID"] = entity.ID.String()
		claims["organizationID"] = entity.OrganizationID.String()
		claims["businessUnitID"] = entity.BusinessUnitID.String()
		claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

		privateKey, _, err := middleware.LoadKeys()
		if err != nil {
			ah.logger.Error().Err(err).Msg("AuthHandler: Error loading keys")
			return err
		}

		t, err := token.SignedString(privateKey)
		if err != nil {
			ah.logger.Error().Err(err).Msg("AuthHandler: Error signing token")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Internal server error",
			})
		}

		// Set the token in a cookie
		c.Cookie(&fiber.Cookie{
			Name:     "trenova-token",
			Value:    t,
			Expires:  time.Now().Add(time.Hour * 72),
			SameSite: "Lax",
			// HTTPOnly: true,
			// Secure:   true,
		})

		return c.JSON(fiber.Map{
			"token": t,
		})
	}
}

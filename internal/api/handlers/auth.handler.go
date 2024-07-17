// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package handlers

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

type AuthenticationHandler struct {
	logger  *zerolog.Logger
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

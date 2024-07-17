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

package middleware

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/utils"
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func Auth(s *server.Server) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{
			JWTAlg: jwtware.RS256,
			Key:    s.Config.Auth.PublicKey,
		},
		TokenLookup:    "cookie:trenova-token",
		ContextKey:     "user",
		SuccessHandler: successHandler(),
		ErrorHandler:   jwtError,
	})
}

func LoadKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKeyData, err := os.ReadFile("private_key.pem")
	if err != nil {
		return nil, nil, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return nil, nil, err
	}

	publicKeyData, err := os.ReadFile("public_key.pem")
	if err != nil {
		return nil, nil, err
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyData)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, publicKey, nil
}

func successHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := extractUserFromContext(c)
		if err != nil {
			return respondWithUnauthorized(c, err.Error())
		}

		claims, err := extractClaims(user)
		if err != nil {
			return respondWithUnauthorized(c, err.Error())
		}

		userID, err := parseAndValidateID(claims, "userID")
		if err != nil {
			return respondWithUnauthorized(c, err.Error())
		}

		orgID, err := parseAndValidateID(claims, "organizationID")
		if err != nil {
			return respondWithUnauthorized(c, err.Error())
		}

		buID, err := parseAndValidateID(claims, "businessUnitID")
		if err != nil {
			return respondWithUnauthorized(c, err.Error())
		}

		setUserContext(c, userID, orgID, buID)

		return c.Next()
	}
}

func extractUserFromContext(c *fiber.Ctx) (*jwt.Token, error) {
	user, ok := c.Locals("user").(*jwt.Token)
	if !ok {
		return nil, errors.New("invalid token: user not found")
	}
	return user, nil
}

func extractClaims(user *jwt.Token) (jwt.MapClaims, error) {
	claims, ok := user.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token: claims not found")
	}
	return claims, nil
}

func parseAndValidateID(claims jwt.MapClaims, key string) (uuid.UUID, error) {
	idStr, ok := claims[key].(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid token: %s not found", key)
	}

	id := uuid.MustParse(idStr)

	return id, nil
}

func setUserContext(c *fiber.Ctx, userID, orgID, buID uuid.UUID) {
	c.Locals(utils.CTXUserID, userID)
	c.Locals(utils.CTXOrganizationID, orgID)
	c.Locals(utils.CTXBusinessUnitID, buID)
}

func respondWithUnauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": message})
}

func jwtError(c *fiber.Ctx, err error) error {
	if err.Error() == "Missing or malformed JWT" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Missing or malformed JWT", "data": nil})
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Invalid or expired JWT", "data": nil})
}

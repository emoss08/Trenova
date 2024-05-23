package middleware

import (
	"crypto/rsa"
	"log"
	"os"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// InitJWT initializes the JWT middleware with the given server configuration
func InitJWT(s *api.Server) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{
			JWTAlg: jwtware.RS256,
			Key:    s.Config.AuthServer.PublicKey,
		},
		TokenLookup: "cookie:trenova-token",
		ContextKey:  "user",
		SuccessHandler: func(c *fiber.Ctx) error {
			user, ok := c.Locals("user").(*jwt.Token)
			if !ok {
				log.Println("Invalid token: user not found")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token: user not found"})
			}

			claims, ok := user.Claims.(jwt.MapClaims)
			if !ok {
				log.Println("Invalid token: claims not found")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token: claims not found"})
			}

			// Extract and validate userID
			userIDStr, ok := claims["userID"].(string)
			if !ok {
				log.Println("Invalid token: userID not found")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token: userID not found"})
			}
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				log.Printf("Invalid token: userID is not a valid UUID: %v", err)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token: userID is not a valid UUID"})
			}

			// Extract and validate organizationID
			orgIDStr, ok := claims["organizationID"].(string)
			if !ok {
				log.Println("Invalid token: organizationID not found")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token: organizationID not found"})
			}
			orgID, err := uuid.Parse(orgIDStr)
			if err != nil {
				log.Printf("Invalid token: organizationID is not a valid UUID: %v", err)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token: organizationID is not a valid UUID"})
			}

			// Extract and validate businessUnitID
			buIDStr, ok := claims["businessUnitID"].(string)
			if !ok {
				log.Println("Invalid token: businessUnitID not found")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token: businessUnitID not found"})
			}
			buID, err := uuid.Parse(buIDStr)
			if err != nil {
				log.Printf("Invalid token: businessUnitID is not a valid UUID: %v", err)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token: businessUnitID is not a valid UUID"})
			}

			// Set user information in context
			c.Locals(util.CTXUserID, userID)
			c.Locals(util.CTXOrganizationID, orgID)
			c.Locals(util.CTXBusinessUnitID, buID)

			return c.Next()
		},
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

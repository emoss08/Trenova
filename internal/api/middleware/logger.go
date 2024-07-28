package middleware

import (
	"time"

	"github.com/emoss08/trenova/config"
	"github.com/gofiber/fiber/v2"
)

// NewCustomFiberzerolog creates a new middleware handler
func NewCustomFiberzerolog(logger *config.ServerLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Handle the request
		err := c.Next()

		// Log the request
		logger.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Str("ip", c.IP()).
			Dur("latency", time.Since(start)).
			Str("user_agent", c.Get("User-Agent")).
			Msg("HTTP Request")

		// Set the user agent in user context
		c.Locals("user-agent", c.Get("User-Agent"))

		return err
	}
}

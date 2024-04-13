package router

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/rs/zerolog/log"
)

// Init initializes the Fiber instance and registers all the routes.
func Init(s *api.Server) {
	s.Fiber = fiber.New(fiber.Config{
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
		// Prefork:     true, // TODO: SET THIS BASED ON A ENV VAR
	})

	apiV1 := s.Fiber.Group("/api")

	// Register the middleware that is globally applied before authentication.
	if s.Config.Fiber.EnableLoggerMiddleware {
		s.Fiber.Use(logger.New())
	} else {
		log.Warn().Msg("Logger middleware is disabled. This is not recommended.")
	}

	if s.Config.Fiber.EnableCORSMiddleware {
		s.Fiber.Use(cors.New(
			cors.Config{
				AllowOrigins:     "https://localhost:5173, http://localhost:5173",
				AllowHeaders:     "Authorization, Origin, Content-Type, Accept, X-CSRF-Token, X-Idempotency-Key",
				AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
				ExposeHeaders:    "Authorization, Origin, Content-Type, Accept, X-CSRF-Token, X-Idempotency-Key",
				AllowCredentials: true,
				MaxAge:           300,
			}))
	} else {
		log.Warn().Msg("CORS middleware is disabled. This is not recommended.")
	}

	// Register the authentication routes.
	auth := apiV1.Group("/auth")
	auth.Post("/login", handlers.AuthenticateUser(s))

	if s.Config.Fiber.EnableHelmetMiddleware {
		apiV1.Use(helmet.New())
	} else {
		log.Warn().Msg("Helmet middleware is disabled. This is not recommended.")
	}

	if s.Config.Fiber.EnableRequestIDMiddleware {
		s.Fiber.Use(requestid.New())
	} else {
		log.Warn().Msg("RequestID middleware is disabled. This is not recommended.")
	}

	if s.Config.Fiber.EnableETagMiddleware {
		s.Fiber.Use(etag.New())
	} else {
		log.Warn().Msg("ETag middleware is disabled. This is not recommended.")
	}

	if s.Config.Fiber.EnableIdempotencyMiddleware {
		apiV1.Use(idempotency.New())
	} else {
		log.Warn().Msg("Idempotency middleware is disabled. This is not recommended.")
	}

	if s.Config.Fiber.EnableSessionMiddleware {
		apiV1.Use(middleware.New(s))
	} else {
		log.Warn().Msg("Session middleware is disabled. This is not recommended.")
	}

	// s.Fiber.Use(recover.New())

	// Health check route.
	s.Fiber.Use(healthcheck.New(healthcheck.Config{
		LivenessEndpoint:  "/live",
		ReadinessEndpoint: "/ready",
	}))

	// Attach all routes.
	handlers.AttachAllRoutes(s, apiV1)
}

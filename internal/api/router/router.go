package router //nolint:cyclop // This package is responsible for setting up the router and registering all the routes.

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/rs/zerolog/log"
)

// Init initializes the Fiber instance and registers all the routes.
// It also registers the middleware that is globally applied before authentication.
//
// Parameters:
//
//	s *api.Server: A pointer to an instance of api.Server which contains configuration and state needed by the router.
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

	// You need to wrap your websocket handler with the websocket.New middleware to handle the upgrade and store the connection.
	s.Fiber.Use("/ws", handlers.NewWebsocketHandler(s.Logger, s.Client).HandleConnection)
	s.Fiber.Get("/ws/:id", websocket.New(handlers.NewWebsocketHandler(s.Logger, s.Client).HandleWebsocketConnection))

	if s.Config.Fiber.EnableMonitorMiddleware {
		// Provide a minimal configuration
		s.Fiber.Get(s.Config.Monitor.Path, monitor.New(
			monitor.Config{
				Title: "Trenova API Metrics",
			},
		))
	}

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

	if s.Config.Fiber.EnableRecoverMiddleware {
		s.Fiber.Use(recover.New())
	} else {
		log.Warn().Msg("Recover middleware is disabled. This is not recommended.")
	}

	if s.Config.Fiber.EnableCompressMiddleware {
		// Initialize default config
		s.Fiber.Use(compress.New())
	} else {
		log.Warn().Msg("Compress middleware is disabled. This is not recommended.")
	}

	if s.Config.Fiber.EnableEncryptCookieMiddleware {
		// Provide a minimal configuration
		s.Fiber.Use(encryptcookie.New(encryptcookie.Config{
			Key: s.Config.Cookie.Key,
		}))
	}

	if s.Config.Fiber.EnableCORSMiddleware {
		s.Fiber.Use(cors.New(
			cors.Config{
				AllowOrigins:     "https://localhost:5173, http://localhost:5173",
				AllowHeaders:     "Authorization, Origin, Content-Type, Accept, X-CSRF-Token, X-Idempotency-Key",
				AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
				AllowCredentials: true,
				MaxAge:           300,
			}))
	} else {
		log.Warn().Msg("CORS middleware is disabled. This is not recommended.")
	}

	// Register the authentication routes.
	auth := apiV1.Group("/auth")
	auth.Post("/login", handlers.NewAuthenticationHandler(s).AuthenticateUser())
	auth.Post("/logout", handlers.NewAuthenticationHandler(s).LogoutUser())
	auth.Post("/check-email", handlers.NewAuthenticationHandler(s).CheckEmail())

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

	// Health check route.
	s.Fiber.Use(healthcheck.New(healthcheck.Config{
		LivenessEndpoint:  "/live",
		ReadinessEndpoint: "/ready",
	}))

	// Attach all routes.
	handlers.AttachAllRoutes(s, apiV1)
}

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

package router

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/handlers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/server"
	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// @title Trenova API
// @version 1.0
// @description This is the API documentation for the Trenova API.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@trenova.app
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:3000
// @BasePath /
func Init(s *server.Server) {
	s.Fiber = fiber.New(fiber.Config{
		JSONEncoder:       sonic.Marshal,
		JSONDecoder:       sonic.Unmarshal,
		Prefork:           s.Config.Fiber.EnablePrefork,
		WriteTimeout:      1 * time.Minute, // 1 minute
		ReadTimeout:       1 * time.Minute, // 1 minute
		EnablePrintRoutes: s.Config.Fiber.EnablePrintRoutes,
	})

	// Initialize the WebsocketHandler
	wsHandler := handlers.NewWebsocketHandler(s)

	// Register the APIv1 routes.
	apiV1 := s.Fiber.Group("/api/v1")

	// Register the middle that is global to all routes.
	if s.Config.Fiber.EnableLoggingMiddleware {
		apiV1.Use(fiberzerolog.New(fiberzerolog.Config{
			Logger: s.Logger,
		}))
	} else {
		log.Warn().Msg("Logging middleware is disabled. This is not recommended.")
	}

	// Register the Prometheus middleware.
	if s.Config.Fiber.EnablePrometheusMiddleware {
		s.Fiber.Use(middleware.PrometheusMiddleware())

		// Endpoint to expose metrics for Prometheus.
		s.Fiber.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	} else {
		log.Warn().Msg("Prometheus middleware is disabled. This is not recommended.")
	}

	// Websocket connection that microservices can use to send messages to the client.
	s.Fiber.Post("/user-tasks/update", handlers.NewUserTaskHandler(s).UpdateTaskStatus)

	// Register the websocket routes.
	s.Fiber.Use("/ws", wsHandler.HandleConnection)
	s.Fiber.Get("/ws/:id", websocket.New(wsHandler.HandleWebsocketConnection))

	// Register the request ID middleware.
	if s.Config.Fiber.EnableRequestIDMiddleware {
		s.Fiber.Use(requestid.New())
	} else {
		log.Warn().Msg("Request ID middleware is disabled. This is not recommended.")
	}

	// Register the helmet middleware.
	if s.Config.Fiber.EnableHelmetMiddleware {
		s.Fiber.Use(helmet.New())
	} else {
		log.Warn().Msg("Helmet middleware is disabled. This is not recommended.")
	}

	// Register the recover middleware.
	if s.Config.Fiber.EnableRecoverMiddleware {
		s.Fiber.Use(recover.New())
	} else {
		log.Warn().Msg("Recover middleware is disabled. This is not recommended.")
	}

	if s.Config.Fiber.EnableCORSMiddleware {
		s.Fiber.Use(cors.New(
			cors.Config{
				AllowOrigins:     s.Config.Cors.AllowedOrigins,
				AllowHeaders:     s.Config.Cors.AllowedHeaders,
				AllowMethods:     s.Config.Cors.AllowedMethods,
				AllowCredentials: s.Config.Cors.AllowCredentials,
				MaxAge:           s.Config.Cors.MaxAge, // Maximum cache age. 3600 = 1 hour
			},
		))
	} else {
		log.Warn().Msg("CORS middleware is disabled. This is not recommended.")
	}

	// Register the authentication routes.
	handlers.NewAuthenticationHandler(s).RegisterRoutes(apiV1)

	// Register the idempotency middleware.
	if s.Config.Fiber.EnableIdempotencyMiddleware {
		apiV1.Use(idempotency.New())
	} else {
		log.Warn().Msg("Idempotency middleware is disabled. This is not recommended.")
	}

	// Register the Authentication middleware
	apiV1.Use(middleware.Auth(s))

	// Register all routes
	handlers.AttachAllRoutes(s, apiV1)

	// cancel the heartbeat on app close.
	s.OnStop("ws.Stop", func(ctx context.Context, app *server.Server) error {
		app.Logger.Info().Msg("Stopping websocket service")
		wsHandler.Stop()
		return nil
	})
}

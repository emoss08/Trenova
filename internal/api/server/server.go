package server

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/telemetry"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/rotisserie/eris"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Lc               fx.Lifecycle
	Config           *config.Config
	Logger           *logger.Logger
	TelemetryMetrics *telemetry.Metrics `name:"telemetryMetrics" optional:"true"`
}

type Server struct {
	app *fiber.App
	cfg *config.Config
	l   *logger.Logger
}

func NewServer(p Params) *Server {
	p.Logger.Debug().
		Bool("telemetry_enabled", p.Config.Telemetry.Enabled).
		Bool("metrics_enabled", p.Config.Telemetry.MetricsEnabled).
		Bool("has_metrics", p.TelemetryMetrics != nil).
		Msg("Server initialization - checking telemetry")
	
	// Create Fiber app with configuration
	app := fiber.New(fiber.Config{
		AppName: fmt.Sprintf(
			"%s v%s",
			p.Config.App.Name,
			p.Config.App.Version,
		),
		JSONEncoder:             sonic.Marshal,
		JSONDecoder:             sonic.Unmarshal,
		BodyLimit:               16 * 1024 * 1024, // 16MB
		ReadBufferSize:          p.Config.Server.ReadBufferSize,
		WriteBufferSize:         p.Config.Server.WriteBufferSize,
		EnableTrustedProxyCheck: p.Config.Server.EnableTrustedProxyCheck,
		ProxyHeader:             p.Config.Server.ProxyHeader,
		// Prefork:                 p.Config.Server.EnablePrefork, // ! Disabled after performance benchmark.
		StreamRequestBody:     p.Config.Server.StreamRequestBody,
		DisableStartupMessage: p.Config.Server.DisableStartupMessage,
		StrictRouting:         p.Config.Server.StrictRouting,
		CaseSensitive:         p.Config.Server.CaseSensitive,
		EnableIPValidation:    p.Config.Server.EnableIPValidation,
		Immutable:             p.Config.Server.Immutable,
		EnablePrintRoutes:     p.Config.Server.EnablePrintRoutes,
		PassLocalsToViews:     p.Config.Server.PassLocalsToViews,
		ErrorHandler:          defaultErrorHandler(p.Logger),
	})

	// Add telemetry middleware if available
	if p.TelemetryMetrics != nil {
		// Create combined middleware with proper configuration
		middleware := telemetry.NewCombinedMiddleware(
			p.Config.Telemetry.ServiceName,
			p.TelemetryMetrics,
			p.Logger,
		)
		
		// Apply all middleware in correct order
		middleware.Apply(app)
		
		p.Logger.Info().Msg("Telemetry middleware added to HTTP server")
	} else {
		// At minimum, add recovery middleware for safety
		app.Use(telemetry.NewRecoveryMiddleware(telemetry.DefaultRecoveryConfig(p.Logger)))
		p.Logger.Debug().Msg("TelemetryMetrics is nil - only recovery middleware added")
	}

	server := &Server{
		app: app,
		cfg: p.Config,
		l:   p.Logger,
	}

	p.Lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return server.Start()
		},
		OnStop: func(ctx context.Context) error {
			return server.Stop(ctx)
		},
	})

	return server
}

func (s *Server) Start() error {
	s.l.Info().Str("listenAddress", s.cfg.Server.ListenAddress).Msg("🚀 HTTP server initialized")

	go func() {
		if err := s.app.Listen(s.cfg.Server.ListenAddress); err != nil {
			s.l.Error().Err(err).Msg("failed to start HTTP server")
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.l.Info().Msg("stopping HTTP server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := s.app.ShutdownWithContext(shutdownCtx)
	if err != nil {
		return eris.Wrap(err, "failed to shutdown HTTP server")
	}

	s.l.Info().Msg("HTTP server stopped")
	return nil
}

func (s *Server) Router() fiber.Router {
	return s.app
}

func defaultErrorHandler(l *logger.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		message := "Internal Server Error"

		var e *fiber.Error
		if eris.As(err, &e) {
			code = e.Code
			message = e.Message
		}

		if eris.Is(err, context.DeadlineExceeded) {
			code = fiber.StatusGatewayTimeout
			message = "Request Timeout"
		}

		// Single consolidated error log with all context
		l.Error().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Str("host", c.Hostname()).
			Str("user-agent", c.Get("User-Agent")).
			Str("referer", c.Get("Referer")).
			Str("accept", c.Get("Accept")).
			Str("content-type", c.Get("Content-Type")).
			Str("content-length", c.Get("Content-Length")).
			Int("code", code).
			Str("message", message).
			Err(err).
			Msg("HTTP request error")

		return c.Status(code).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    code,
				"message": message,
			},
		})
	}
}

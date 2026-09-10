//revive:disable-next-line:var-naming
package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Config       *config.Config
	Logger       *zap.Logger
	ErrorHandler *helpers.ErrorHandler
	LC           fx.Lifecycle
}

type Server struct {
	httpServer *http.Server
	router     *gin.Engine
	cfg        *config.Config
	l          *zap.Logger
}

func NewServer(p Params) (*Server, error) {
	gin.SetMode(p.Config.Server.Mode)

	router := gin.New()
	if err := configureClientIPResolution(router, p.Config); err != nil {
		return nil, err
	}

	handler := middleware.NewRequestTimeoutHandler(router, p.Config, p.ErrorHandler)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", p.Config.Server.Host, p.Config.Server.Port),
		Handler:           handler,
		ReadTimeout:       p.Config.Server.ReadTimeout,
		ReadHeaderTimeout: p.Config.Server.ReadHeaderTimeout,
		WriteTimeout:      p.Config.Server.WriteTimeout,
		IdleTimeout:       p.Config.Server.IdleTimeout,
	}

	server := &Server{
		router:     router,
		l:          p.Logger,
		cfg:        p.Config,
		httpServer: httpServer,
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return server.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return server.Stop(ctx)
		},
	})

	return server, nil
}

func configureClientIPResolution(router *gin.Engine, cfg *config.Config) error {
	router.TrustedPlatform = trustedPlatformHeader(cfg.Server.TrustedPlatform)

	if err := router.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		return fmt.Errorf("configure trusted proxies: %w", err)
	}

	return nil
}

func trustedPlatformHeader(platform string) string {
	switch platform {
	case config.TrustedPlatformCloudflare:
		return gin.PlatformCloudflare
	case config.TrustedPlatformGoogleAppEngine:
		return gin.PlatformGoogleAppEngine
	case config.TrustedPlatformFlyIO:
		return gin.PlatformFlyIO
	default:
		return ""
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.l.Info(
		"Starting HTTP server",
		zap.String("address", fmt.Sprintf("http://%s", s.httpServer.Addr)),
		zap.String("mode", s.cfg.Server.Mode),
	)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.l.Fatal("failed to start HTTP server", zap.Error(err))
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.l.Info("Stopping HTTP server")
	return s.httpServer.Shutdown(ctx)
}

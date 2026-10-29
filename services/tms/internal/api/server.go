//revive:disable-next-line:var-naming
package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/serverconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	AppConfig    *appconfig.Config
	ServerConfig *serverconfig.Config
	Logger       *zap.Logger
	ErrorHandler *helpers.ErrorHandler
	LC           fx.Lifecycle
	// Realtime ends its event streams as soon as shutdown begins. Shutdown
	// otherwise waits on every open stream until its deadline, since a stream
	// is a request that never finishes on its own.
	Realtime realtimeports.RealtimeGateway `optional:"true"`
}

type Server struct {
	httpServer *http.Server
	router     *gin.Engine
	cfg        *serverconfig.Config
	l          *zap.Logger
}

func NewServer(p Params) (*Server, error) {
	gin.SetMode(p.ServerConfig.Mode)

	router := gin.New()
	if err := configureClientIPResolution(router, p.ServerConfig); err != nil {
		return nil, err
	}

	handler := middleware.NewRequestTimeoutHandler(
		router,
		p.AppConfig,
		p.ServerConfig,
		p.ErrorHandler,
	)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", p.ServerConfig.Host, p.ServerConfig.Port),
		Handler:           handler,
		ReadTimeout:       p.ServerConfig.ReadTimeout,
		ReadHeaderTimeout: p.ServerConfig.ReadHeaderTimeout,
		WriteTimeout:      p.ServerConfig.WriteTimeout,
		IdleTimeout:       p.ServerConfig.IdleTimeout,
	}
	if p.Realtime != nil {
		httpServer.RegisterOnShutdown(p.Realtime.Drain)
	}

	server := &Server{
		router:     router,
		l:          p.Logger,
		cfg:        p.ServerConfig,
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

func configureClientIPResolution(router *gin.Engine, cfg *serverconfig.Config) error {
	router.TrustedPlatform = trustedPlatformHeader(cfg.TrustedPlatform)

	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return fmt.Errorf("configure trusted proxies: %w", err)
	}

	return nil
}

func trustedPlatformHeader(platform string) string {
	switch platform {
	case serverconfig.TrustedPlatformCloudflare:
		return gin.PlatformCloudflare
	case serverconfig.TrustedPlatformGoogleAppEngine:
		return gin.PlatformGoogleAppEngine
	case serverconfig.TrustedPlatformFlyIO:
		return gin.PlatformFlyIO
	default:
		return ""
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.l.Info(
		"Starting HTTP server",
		zap.String("address", fmt.Sprintf("http://%s", s.httpServer.Addr)),
		zap.String("mode", s.cfg.Mode),
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

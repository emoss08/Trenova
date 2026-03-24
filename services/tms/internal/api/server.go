package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
	LC     fx.Lifecycle
}

type Server struct {
	httpServer *http.Server
	router     *gin.Engine
	cfg        *config.Config
	l          *zap.Logger
}

func NewServer(p Params) *Server {
	gin.SetMode(p.Config.Server.Mode)

	router := gin.New()

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", p.Config.Server.Host, p.Config.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
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

	return server
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

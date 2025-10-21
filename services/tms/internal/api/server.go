package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
		Addr:        fmt.Sprintf("%s:%d", p.Config.Server.Host, p.Config.Server.Port),
		Handler:     router,
		ReadTimeout: 30 * time.Second,
		// WriteTimeout set to 0 to support long-lived SSE connections
		// Individual handlers can still enforce their own timeouts via context
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
		OnStart: func(context.Context) error {
			return server.Start()
		},
		OnStop: func(context.Context) error {
			return server.Stop()
		},
	})

	return server
}

func (s *Server) Start() error {
	s.l.Info("Starting HTTP server",
		zap.String("address", s.httpServer.Addr),
		zap.String("mode", s.cfg.Server.Mode),
	)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.l.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		s.l.Info("Received shutdown signal")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.l.Error("Error during server shutdown", zap.Error(err))
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	s.l.Info("Stopping HTTP server")
	return s.httpServer.Shutdown(context.Background())
}

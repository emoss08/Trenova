package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/emoss08/trenova/shared/edi/internal/bootstrap"
	"github.com/emoss08/trenova/shared/edi/internal/infrastructure/repositories"
	httpTransport "github.com/emoss08/trenova/shared/edi/internal/transport/http"
	"github.com/emoss08/trenova/shared/edi/internal/core/services"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: .env file not found: %v\n", err)
	}

	app := fx.New(
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger}
		}),
		bootstrap.Module,
		repositories.Module,
		services.Module,
		fx.Provide(
			ProvideServerConfig,
			httpTransport.NewHTTPServer,
		),
		fx.Invoke(RegisterHTTPServer),
	)

	app.Run()
}

// ProvideServerConfig provides the HTTP server configuration
func ProvideServerConfig() *httpTransport.ServerConfig {
	config := httpTransport.DefaultServerConfig()
	
	// Override from environment
	if port := os.Getenv("SERVICE_PORT"); port != "" {
		config.Address = ":" + port
	}
	
	if os.Getenv("ENVIRONMENT") == "development" {
		config.EnableDetailedLogging = true
	}
	
	return &config
}

// RegisterHTTPServer registers the HTTP server with fx lifecycle
func RegisterHTTPServer(lifecycle fx.Lifecycle, server *http.Server, logger *zap.Logger) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("starting HTTP server", 
				zap.String("address", server.Addr),
				zap.String("service", "edi-processor"),
			)
			
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Fatal("failed to start HTTP server", zap.Error(err))
				}
			}()
			
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping HTTP server")
			return server.Shutdown(ctx)
		},
	})
}
package bootstrap

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/trenova-app/transport/internal/bootstrap/modules/api"
	"github.com/trenova-app/transport/internal/bootstrap/modules/infrastructure"
	"github.com/trenova-app/transport/internal/bootstrap/modules/services"
	"github.com/trenova-app/transport/internal/bootstrap/modules/validators"
	redisRepos "github.com/trenova-app/transport/internal/infrastructure/cache/redis/repositories"
	postgresRepos "github.com/trenova-app/transport/internal/infrastructure/database/postgres/repositories"

	"go.uber.org/fx"
)

// Bootstrap initializes and starts the application
func Bootstrap() error {
	app := fx.New(
		infrastructure.Module,
		redisRepos.Module,
		postgresRepos.Module,
		validators.Module,
		services.Module,
		api.Module,
	)

	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.Start(startCtx); err != nil {
		return err
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return app.Stop(stopCtx)
}

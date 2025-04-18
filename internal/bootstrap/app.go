package bootstrap

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/trenova/internal/bootstrap/modules/api"
	"github.com/emoss08/trenova/internal/bootstrap/modules/external"
	"github.com/emoss08/trenova/internal/bootstrap/modules/infrastructure"
	"github.com/emoss08/trenova/internal/bootstrap/modules/services"
	"github.com/emoss08/trenova/internal/bootstrap/modules/validators"
	"github.com/emoss08/trenova/internal/core/services/analytics"
	redisRepos "github.com/emoss08/trenova/internal/infrastructure/cache/redis/repositories"
	postgresRepos "github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/statemachine"

	"go.uber.org/fx"
)

// Bootstrap initializes and starts the application
func Bootstrap() error {
	app := fx.New(
		infrastructure.Module,
		infrastructure.BackupModule,
		redisRepos.Module,
		statemachine.Module,
		services.CalculatorModule,
		postgresRepos.Module,
		external.Module,
		validators.Module,
		analytics.Module,
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

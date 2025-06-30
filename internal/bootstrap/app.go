package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/trenova/internal/bootstrap/modules/api"
	"github.com/emoss08/trenova/internal/bootstrap/modules/external"
	"github.com/emoss08/trenova/internal/bootstrap/modules/infrastructure"
	"github.com/emoss08/trenova/internal/bootstrap/modules/seqgen"
	"github.com/emoss08/trenova/internal/bootstrap/modules/services"
	"github.com/emoss08/trenova/internal/bootstrap/modules/validators"
	"github.com/emoss08/trenova/internal/core/services/analytics"
	"github.com/emoss08/trenova/internal/core/services/streaming"
	redisRepos "github.com/emoss08/trenova/internal/infrastructure/cache/redis/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/cdc"
	postgresRepos "github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/jobs"
	"github.com/emoss08/trenova/internal/pkg/statemachine"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// Bootstrap initializes and starts the application
func Bootstrap() error {
	app := fx.New(
		infrastructure.Module,
		infrastructure.BackupModule,
		redisRepos.Module,
		statemachine.Module,
		seqgen.Module,
		services.CalculatorModule,
		postgresRepos.Module,
		external.Module,
		cdc.Module,
		validators.Module,
		analytics.Module,
		services.Module,
		streaming.Module,
		api.Module,
		jobs.Module,
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ZapLogger{Logger: zap.NewExample()}
		}),
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

	fmt.Println("Shutdown initiated, closing resources...")

	// Graceful shutdown with a deadline warning
	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set up a deadline warning
	go func() {
		select {
		case <-stopCtx.Done():
			// Context deadline exceeded, but we still want to continue shutdown
			fmt.Println(
				"WARNING: Shutdown is taking longer than expected, some resources may not be properly cleaned up",
			)
		case <-time.After(5 * time.Second):
			// This will only trigger if stopCtx doesn't finish within 5 seconds
			fmt.Println("Shutdown in progress, waiting for resources to clean up...")
		}
	}()

	err := app.Stop(stopCtx)
	if err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
		// Even if we have an error, we return nil to ensure the process exits cleanly
		return nil
	}

	fmt.Println("Shutdown completed successfully")
	return nil
}

/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
	"github.com/emoss08/trenova/internal/infrastructure/encryption"
	"github.com/emoss08/trenova/internal/infrastructure/jobs"
	"github.com/emoss08/trenova/internal/infrastructure/telemetry"
	"github.com/emoss08/trenova/internal/pkg/email"
	"github.com/emoss08/trenova/internal/pkg/formula"
	"github.com/emoss08/trenova/internal/pkg/statemachine"
	"go.uber.org/fx"
)

// Bootstrap initializes and starts the application
func Bootstrap() error {
	app := fx.New(
		infrastructure.ConfigModule,
		infrastructure.LoggerModule,
		telemetry.Module,
		infrastructure.DatabaseModule,
		infrastructure.StorageModule,
		infrastructure.CacheModule,
		infrastructure.BackupModule,
		redisRepos.Module,
		statemachine.Module,
		seqgen.Module,
		formula.Module,
		services.CalculatorModule,
		postgresRepos.Module,
		external.Module,
		cdc.Module,
		encryption.Module,
		validators.Module,
		analytics.Module,
		email.Module,
		services.Module,
		streaming.Module,
		api.Module,
		jobs.Module,
	)

	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.Start(startCtx); err != nil {
		return err
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutdown initiated, closing resources...")

	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		select {
		case <-stopCtx.Done():
			fmt.Println(
				"WARNING: Shutdown is taking longer than expected, some resources may not be properly cleaned up",
			)
		case <-time.After(5 * time.Second):
			fmt.Println("Shutdown in progress, waiting for resources to clean up...")
		}
	}()

	err := app.Stop(stopCtx)
	if err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
		return nil
	}

	fmt.Println("Shutdown completed successfully")
	return nil
}

package postgres

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"go.uber.org/zap"
)

const (
	maintenanceStatementBackstop = 30 * time.Minute
	maintenanceDebugEnv          = "BUNDEBUG"
	maintenanceDebugOff          = "0"
	maintenanceDebugVerbose      = "2"
)

func OpenMaintenanceDB(ctx context.Context, cfg *config.Config) (*keel.DB, error) {
	observers, err := maintenanceObservers(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := dbpool.Open(ctx, &dbpool.Options{
		DSN: cfg.GetMigrationDSN(),
		RuntimeParams: map[string]string{
			"statement_timeout": fmt.Sprintf("%dms", maintenanceStatementBackstop.Milliseconds()),
			"lock_timeout": fmt.Sprintf(
				"%dms",
				max(cfg.Database.GetLockTimeout().Milliseconds(), 1),
			),
		},
		Observers: observers,
	})
	if err != nil {
		return nil, fmt.Errorf("open the maintenance connection: %w", err)
	}

	return pool.Keel, nil
}

func maintenanceObservers(cfg *config.Config) ([]keel.Observer, error) {
	mode := os.Getenv(maintenanceDebugEnv)
	if !cfg.App.Debug || mode == maintenanceDebugOff {
		return nil, nil
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("build the maintenance query logger: %w", err)
	}

	return []keel.Observer{&debugObserver{
		logger:  observability.NewContextLogger(logger).Named("keel"),
		verbose: cfg.Database.Verbose || mode == maintenanceDebugVerbose,
	}}, nil
}

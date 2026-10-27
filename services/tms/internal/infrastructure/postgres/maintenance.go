package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/keeltables"
)

const (
	maintenanceStatementBackstop = 30 * time.Minute
)

func OpenMaintenanceDB(ctx context.Context, cfg *config.Config) (*keel.DB, error) {
	var observers []keel.Observer
	if log := queryLog(cfg); log != nil {
		observers = append(observers, log)
	}

	db, err := dbpool.Open(ctx, &dbpool.Options{
		DSN: cfg.GetMigrationDSN(),
		RuntimeParams: map[string]string{
			"statement_timeout": fmt.Sprintf("%dms", maintenanceStatementBackstop.Milliseconds()),
			"lock_timeout": fmt.Sprintf(
				"%dms",
				max(cfg.Database.GetLockTimeout().Milliseconds(), 1),
			),
		},
		Observers:   observers,
		ObserveArgs: len(observers) > 0,
		Tables:      keeltables.All(),
	})
	if err != nil {
		return nil, fmt.Errorf("open the maintenance connection: %w", err)
	}

	return db, nil
}

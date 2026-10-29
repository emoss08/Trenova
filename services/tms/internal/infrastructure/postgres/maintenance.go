package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/keeltables"
)

const (
	maintenanceStatementBackstop = 30 * time.Minute
)

func OpenMaintenanceDB(
	ctx context.Context,
	appConfig *appconfig.Config,
	databaseConfig *databaseconfig.Config,
) (*keel.DB, error) {
	var observers []keel.Observer
	if log := queryLog(appConfig, databaseConfig); log != nil {
		observers = append(observers, log)
	}

	db, err := dbpool.Open(ctx, &dbpool.Options{
		DSN: databaseConfig.MigrationDSN(appConfig.Name),
		RuntimeParams: map[string]string{
			"statement_timeout": fmt.Sprintf("%dms", maintenanceStatementBackstop.Milliseconds()),
			"lock_timeout": fmt.Sprintf(
				"%dms",
				max(databaseConfig.GetLockTimeout().Milliseconds(), 1),
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

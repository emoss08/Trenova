package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
)

const maintenanceStatementBackstop = 30 * time.Minute

func OpenMaintenanceDB(ctx context.Context, cfg *config.Config) (*bun.DB, error) {
	pool, err := dbpool.Open(ctx, &dbpool.Options{
		DSN: cfg.GetMigrationDSN(),
		RuntimeParams: map[string]string{
			"statement_timeout": fmt.Sprintf("%dms", maintenanceStatementBackstop.Milliseconds()),
			"lock_timeout": fmt.Sprintf(
				"%dms",
				max(cfg.Database.GetLockTimeout().Milliseconds(), 1),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open the maintenance connection: %w", err)
	}

	db := pool.Bun
	if cfg.App.Debug {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.FromEnv("BUNDEBUG")))
	}
	db.RegisterModel(domainregistry.RegisterManyToManyEntities()...)
	db.RegisterModel(domainregistry.RegisterEntities()...)

	return db, nil
}

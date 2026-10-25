package dbcli

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/migrate"
	"github.com/emoss08/trenova/pkg/dbscope"
)

const migratedTablesReason = "record the tables a migration run changed"

func markMigratedTables(
	config func() (keel.Config, error),
) func(context.Context, []migrate.Migration) error {
	return func(ctx context.Context, _ []migrate.Migration) error {
		cfg, err := config()
		if err != nil {
			return err
		}

		db, err := keel.Open(ctx, cfg)
		if err != nil {
			return fmt.Errorf("open the database to record migrated tables: %w", err)
		}
		defer db.Close()

		names := make([]string, 0, len(cfg.Tables))
		for _, table := range cfg.Tables {
			names = append(names, table.Name)
		}

		keel.MarkChanged(dbscope.WithSystem(ctx, migratedTablesReason), db, names...)

		if err = db.FlushDataVersions(ctx); err != nil {
			return fmt.Errorf("record migrated tables: %w", err)
		}

		return nil
	}
}

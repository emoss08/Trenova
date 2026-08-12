package migrator

import (
	pgmigrations "github.com/emoss08/trenova/internal/infrastructure/postgres/migrations"
	sqlitemigrations "github.com/emoss08/trenova/internal/infrastructure/sqlite/migrations"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/migrate"
)

func migrationsFor(db *bun.DB) *migrate.Migrations {
	if db == nil {
		return pgmigrations.Migrations
	}

	if db.Dialect().Name() == dialect.SQLite {
		return sqlitemigrations.Migrations
	}

	return pgmigrations.Migrations
}

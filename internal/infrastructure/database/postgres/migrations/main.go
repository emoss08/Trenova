// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package migrations

import (
	"context"
	"embed"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

var Migrations = migrate.NewMigrations()

//go:embed *.sql
var sqlMigrations embed.FS

func init() {
	if err := Migrations.Discover(sqlMigrations); err != nil {
		panic(err)
	}
}

// Setup initializes migrations with proper configuration
func Setup() *migrate.Migrations {
	migrations := migrate.NewMigrations()

	if err := migrations.Discover(sqlMigrations); err != nil {
		panic(fmt.Errorf("failed to discover migrations: %w", err))
	}

	return migrations
}

// Run executes all pending migrations
func Run(ctx context.Context, db *bun.DB) error {
	migrations := Setup()

	migrator := migrate.NewMigrator(db, migrations)

	// Create migrations table if it doesn't exist
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to init migrator: %w", err)
	}

	// Run all pending migrations
	group, err := migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if group.ID == 0 {
		fmt.Printf("no new migrations to run\n")
		return nil
	}

	fmt.Printf("migrated to %d\n", group.ID)
	return nil
}

// Reset drops all tables and re-runs migrations
func Reset(ctx context.Context, db *bun.DB) error {
	migrations := Setup()

	migrator := migrate.NewMigrator(db, migrations)

	// Create migrations table if it doesn't exist
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to init migrator: %w", err)
	}

	// Drop all tables
	if err := migrator.Reset(ctx); err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}

	// Run all migrations
	group, err := migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Printf("migrated to %d\n", group.ID)
	return nil
}

/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"

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

// Fingerprint returns a stable hex digest of every embedded migration file.
// Test harnesses use it to key a cached, pre-migrated template database so the
// cache is invalidated automatically whenever a migration is added or edited.
func Fingerprint() (string, error) {
	entries, err := fs.ReadDir(sqlMigrations, ".")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	digest := sha256.New()
	for _, name := range names {
		contents, readErr := sqlMigrations.ReadFile(name)
		if readErr != nil {
			return "", fmt.Errorf("failed to read migration %q: %w", name, readErr)
		}
		digest.Write([]byte(name))
		digest.Write(contents)
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
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

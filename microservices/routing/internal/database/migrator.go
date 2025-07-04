package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// Migrator handles database migrations
type Migrator struct {
	db     *sql.DB
	logger zerolog.Logger
}

// NewMigrator creates a new migrator instance
func NewMigrator(dsn string, logger zerolog.Logger) (*Migrator, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &Migrator{
		db:     db,
		logger: logger,
	}, nil
}

// Migrate runs all pending migrations
func (m *Migrator) Migrate(ctx context.Context) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting dialect: %w", err)
	}

	// Run migrations
	if err := goose.UpContext(ctx, m.db, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	// Get current version
	version, err := goose.GetDBVersionContext(ctx, m.db)
	if err != nil {
		return fmt.Errorf("getting db version: %w", err)
	}

	m.logger.Info().Int64("version", version).Msg("Database migrated successfully")
	return nil
}

// Status returns the current migration status
func (m *Migrator) Status(ctx context.Context) error {
	return goose.StatusContext(ctx, m.db, "migrations")
}

// Close closes the database connection
func (m *Migrator) Close() error {
	return m.db.Close()
}
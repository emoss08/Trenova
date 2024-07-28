// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/migrate/migrations"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
)

type TestDBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func getDBConfig() TestDBConfig {
	return TestDBConfig{
		Host:     getEnv("TEST_DB_HOST", "localhost"),
		Port:     getEnv("TEST_DB_PORT", "5432"),
		User:     getEnv("TEST_DB_USER", "postgres"),
		Password: getEnv("TEST_DB_PASSWORD", "postgres"),
		DBName:   getEnv("TEST_DB_NAME", "test_db"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// TestDB encapsulates the database connection and provides utility methods
type TestDB struct {
	DB   *bun.DB
	once sync.Once
}

// NewTestDB creates a new test database connection
func NewTestDB() (*TestDB, error) {
	config := getDBConfig()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.User, config.Password, config.Host, config.Port, config.DBName)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	// Set connection pool parameters
	sqldb.SetMaxOpenConns(4)
	sqldb.SetMaxIdleConns(4)
	sqldb.SetConnMaxLifetime(time.Hour)

	db := bun.NewDB(sqldb, pgdialect.New())

	// Add query hook for debugging
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.FromEnv("BUNDEBUG"),
	))

	return &TestDB{DB: db}, nil
}

// GetDB returns the bun.DB instance
func (tdb *TestDB) GetDB() *bun.DB {
	return tdb.DB
}

// InitSchema initializes the database schema
func (tdb *TestDB) InitSchema() error {
	var err error
	tdb.once.Do(func() {
		ctx := context.Background()
		migrator := migrate.NewMigrator(tdb.DB, migrations.Migrations)

		// Initialize the migration tables
		if err = migrator.Init(ctx); err != nil {
			err = fmt.Errorf("failed to initialize migrator: %v", err)
			return
		}

		if err = migrator.Lock(ctx); err != nil {
			err = fmt.Errorf("failed to acquire migration lock: %v", err)
			return
		}
		defer migrator.Unlock(ctx)

		group, migrateErr := migrator.Migrate(ctx)
		if migrateErr != nil {
			err = fmt.Errorf("failed to run migrations: %v", migrateErr)
			return
		}

		if group.IsZero() {
			fmt.Println("There are no new migrations to run (database is up to date)")
		} else {
			fmt.Printf("Applied %d migrations\n", len(group.Migrations))
		}
	})
	return err
}

func (tdb *TestDB) ResetDatabase() error {
	ctx := context.Background()

	// Drop all tables, including migration tables
	if err := tdb.DropAllTablesAndTypes(ctx); err != nil {
		return fmt.Errorf("failed to drop tables and types: %w", err)
	}

	// Recreate migration tables
	migrator := migrate.NewMigrator(tdb.DB, migrations.Migrations)
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}

	// Run migrations to recreate schema
	group, err := migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if group.IsZero() {
		fmt.Println("No migrations were applied")
	} else {
		fmt.Printf("Applied %d migrations\n", len(group.Migrations))
	}

	return nil
}

func (tdb *TestDB) DropAllTablesAndTypes(ctx context.Context) error {
	query := `
	DO $$ 
	DECLARE 
		r RECORD;
	BEGIN
		-- Disable triggers
		EXECUTE 'SET session_replication_role = replica';

		-- Drop all tables
		FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
			EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
		END LOOP;

		-- Drop all custom types
		FOR r IN (SELECT typname FROM pg_type WHERE typtype = 'e' AND typnamespace = (SELECT oid FROM pg_namespace WHERE nspname = current_schema())) LOOP
			EXECUTE 'DROP TYPE IF EXISTS ' || quote_ident(r.typname) || ' CASCADE';
		END LOOP;

		-- Re-enable triggers
		EXECUTE 'SET session_replication_role = DEFAULT';
	END $$;
	`

	_, err := tdb.DB.ExecContext(ctx, query)
	return err
}

func (tdb *TestDB) Close() error {
	return tdb.DB.Close()
}

// WithTransaction runs a function within a transaction
func (tdb *TestDB) WithTransaction(fn func(*bun.Tx) error) error {
	return tdb.DB.RunInTx(context.Background(), nil, func(_ context.Context, tx bun.Tx) error {
		return fn(&tx)
	})
}

// SetupTestCase prepares the database for a test case
// SetupTestCase prepares the database for a test case
func SetupTestCase(t *testing.T) (*TestDB, func()) {
	t.Helper()

	testDB, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Reset the database to a clean state
	if err = testDB.ResetDatabase(); err != nil {
		t.Fatalf("Failed to reset database: %v", err)
	}

	return testDB, func() {
		if err = testDB.Close(); err != nil {
			t.Errorf("Failed to close database connection: %v", err)
		}
	}
}

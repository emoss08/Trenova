package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/emoss08/trenova/internal/pkg/registry"
	"github.com/emoss08/trenova/internal/pkg/utils/fileutils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
)

const (
	dbName     = "test_db"
	dbUser     = "test_user"
	dbPassword = "test_password"
	dbPort     = "5432/tcp"
	timeout    = 60 * time.Second
)

type TestDatabase struct {
	Container testcontainers.Container
	DB        *bun.DB
	Fixture   *dbfixture.Fixture
	Migrator  *migrate.Migrator
	t         testing.TB
}

func NewTestDatabase(t testing.TB, migrations *migrate.Migrations) *TestDatabase {
	t.Helper()

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{dbPort},
		Env: map[string]string{
			"POSTGRES_DB":       dbName,
			"POSTGRES_USER":     dbUser,
			"POSTGRES_PASSWORD": dbPassword,
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(nat.Port(dbPort)),
			wait.ForLog("database system is ready to accept connections"),
		).WithDeadline(timeout),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(dbPort))
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser,
		dbPassword,
		host,
		mappedPort.Port(),
		dbName,
	)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("failed to parse database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create database pool: %v", err)
	}

	sqldb := stdlib.OpenDBFromPool(pool)
	bunDB := bun.NewDB(sqldb, pgdialect.New())

	bunDB.AddQueryHook(
		bundebug.NewQueryHook(bundebug.WithVerbose(false), bundebug.FromEnv("BUNDEBUG")),
	)

	// Register entities
	bunDB.RegisterModel(registry.RegisterEntities()...)

	// Create migrator
	migrator := migrate.NewMigrator(bunDB, migrations)

	// Initialize fixture
	// helpers := fixtures.NewFixtureHelpers()
	// fixture := dbfixture.New(bunDB, dbfixture.WithTemplateFuncs(helpers.GetTemplateFuncs()))

	testDB := &TestDatabase{
		Container: container,
		DB:        bunDB,
		// Fixture:   fixture,
		Migrator: migrator,
		t:        t,
	}

	if err := testDB.InitializeDB(ctx); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	return testDB
}

// InitializeDB sets up the database schema and initial state
func (td *TestDatabase) InitializeDB(ctx context.Context) error {
	// Reset database schema
	// if err := td.ResetSchema(ctx); err != nil {
	// 	return fmt.Errorf("failed to reset schema: %w", err)
	// }

	// Initialize and run migrations
	if err := td.Migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to init migrations: %w", err)
	}

	if err := td.Migrator.Lock(ctx); err != nil {
		return fmt.Errorf("failed to lock migrations: %w", err)
	}

	if _, err := td.Migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := td.Migrator.Unlock(ctx); err != nil {
		return fmt.Errorf("failed to unlock migrations: %w", err)
	}

	return nil
}

// ResetSchema resets the database schema
func (td *TestDatabase) ResetSchema(ctx context.Context) error {
	_, err := td.DB.ExecContext(ctx, `
        DROP SCHEMA IF EXISTS public CASCADE;
        CREATE SCHEMA public;
        GRANT ALL ON SCHEMA public TO postgres;
        GRANT ALL ON SCHEMA public TO public;
    `)
	if err != nil {
		return fmt.Errorf("failed to reset schema: %w", err)
	}
	return nil
}

// LoadFixtures loads fixture data from the specified directory
func (td *TestDatabase) LoadFixtures(ctx context.Context) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	projectRoot, err := fileutils.FindProjectRoot(workingDir)
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	fixturesPath := filepath.Join(projectRoot, "test", "fixtures")

	if _, err := os.Stat(fixturesPath); os.IsNotExist(err) {
		return fmt.Errorf("fixtures directory does not exist: %s", fixturesPath)
	}

	if err = fileutils.EnsureDirExists(fixturesPath); err != nil {
		return fmt.Errorf("failed to ensure fixtures directory exists: %w", err)
	}

	if err = td.Fixture.Load(ctx, os.DirFS(fixturesPath), "fixtures.yml"); err != nil {
		return fmt.Errorf("failed to load fixtures: %w", err)
	}

	return nil
}

// Cleanup closes the database connection and removes the container
func (td *TestDatabase) Cleanup() {
	td.t.Helper()

	ctx := context.Background()

	if td.DB != nil {
		if err := td.DB.Close(); err != nil {
			td.t.Errorf("failed to close database connection: %s", err)
		}
	}

	if td.Container != nil {
		if err := td.Container.Terminate(ctx); err != nil {
			td.t.Errorf("failed to terminate container: %s", err)
		}
	}
}

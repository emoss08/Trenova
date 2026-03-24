package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var (
	sharedContainer *PostgresContainer
	sharedOnce      sync.Once
	sharedErr       error
)

type PostgresContainer struct {
	container *postgres.PostgresContainer
	dsn       string
	db        *bun.DB
}

type containerWrapper struct {
	container *postgres.PostgresContainer
}

func (w *containerWrapper) Terminate(ctx context.Context) error {
	return w.container.Terminate(ctx)
}

type PostgresOptions struct {
	Database string
	Username string
	Password string
	Image    string
}

func DefaultPostgresOptions() PostgresOptions {
	return PostgresOptions{
		Database: "trenova_test",
		Username: "test",
		Password: "test",
		Image:    "postgres:16-alpine",
	}
}

func SetupPostgres(
	t *testing.T,
	tc *TestContext,
	opts ...func(*PostgresOptions),
) *PostgresContainer {
	t.Helper()

	options := DefaultPostgresOptions()
	for _, opt := range opts {
		opt(&options)
	}

	container, err := postgres.Run(tc.Ctx,
		options.Image,
		postgres.WithDatabase(options.Database),
		postgres.WithUsername(options.Username),
		postgres.WithPassword(options.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	require.NoError(t, err, "failed to start postgres container")

	tc.AddContainer(&containerWrapper{container: container})

	dsn, err := container.ConnectionString(tc.Ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	err = db.PingContext(tc.Ctx)
	require.NoError(t, err, "failed to ping database")

	return &PostgresContainer{
		container: container,
		dsn:       dsn,
		db:        db,
	}
}

func (p *PostgresContainer) DB() *bun.DB {
	return p.db
}

func (p *PostgresContainer) DSN() string {
	return p.dsn
}

func (p *PostgresContainer) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *PostgresContainer) Terminate(ctx context.Context) error {
	if p.db != nil {
		p.db.Close()
	}
	if p.container != nil {
		return p.container.Terminate(ctx)
	}
	return nil
}

func (p *PostgresContainer) Exec(ctx context.Context, query string) error {
	_, err := p.db.ExecContext(ctx, query)
	return err
}

func (p *PostgresContainer) TruncateAll(ctx context.Context) error {
	query := `
		DO $$
		DECLARE
			r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
				EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;
	`
	return p.Exec(ctx, query)
}

func WithDatabase(name string) func(*PostgresOptions) {
	return func(o *PostgresOptions) {
		o.Database = name
	}
}

func WithImage(image string) func(*PostgresOptions) {
	return func(o *PostgresOptions) {
		o.Image = image
	}
}

func WithCredentials(username, password string) func(*PostgresOptions) {
	return func(o *PostgresOptions) {
		o.Username = username
		o.Password = password
	}
}

func RequireIntegration(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

func getSharedPostgres() (*PostgresContainer, error) {
	sharedOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		options := DefaultPostgresOptions()

		container, err := postgres.Run(ctx,
			options.Image,
			postgres.WithDatabase(options.Database),
			postgres.WithUsername(options.Username),
			postgres.WithPassword(options.Password),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2),
			),
			testcontainers.WithReuseByName("trenova-test-postgres"),
		)
		if err != nil {
			sharedErr = fmt.Errorf("failed to start postgres container: %w", err)
			return
		}

		host, err := container.Host(ctx)
		if err != nil {
			sharedErr = fmt.Errorf("failed to get postgres host: %w", err)
			return
		}

		port, err := container.MappedPort(ctx, "5432")
		if err != nil {
			sharedErr = fmt.Errorf("failed to get postgres port: %w", err)
			return
		}

		adminDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			options.Username, options.Password, host, port.Port(), options.Database)
		adminSQL := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(adminDSN)))
		adminDB := bun.NewDB(adminSQL, pgdialect.New())

		for i := range 30 {
			if pingErr := adminDB.PingContext(ctx); pingErr == nil {
				break
			}
			if i == 29 {
				adminDB.Close()
				sharedErr = fmt.Errorf("failed to connect to postgres container after retries")
				return
			}
			time.Sleep(500 * time.Millisecond)
		}

		dbName := fmt.Sprintf("trenova_test_%d", os.Getpid())

		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		_, err = adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
		adminDB.Close()
		if err != nil {
			sharedErr = fmt.Errorf("failed to create per-process database: %w", err)
			return
		}

		dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			options.Username, options.Password, host, port.Port(), dbName)
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		if err := db.PingContext(ctx); err != nil {
			sharedErr = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		sharedContainer = &PostgresContainer{
			container: container,
			dsn:       dsn,
			db:        db,
		}
	})

	return sharedContainer, sharedErr
}

func SetupTestDB(t *testing.T) (*TestContext, *bun.DB) {
	t.Helper()
	RequireIntegration(t)

	pg, err := getSharedPostgres()
	require.NoError(t, err, "failed to get shared postgres container")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	if err := pg.TruncateAll(ctx); err != nil {
		t.Logf("Warning: failed to truncate tables: %v", err)
	}

	tc := &TestContext{
		T:          t,
		Ctx:        ctx,
		Cancel:     cancel,
		Containers: make([]Container, 0),
	}
	t.Cleanup(func() {
		cancel()
	})

	return tc, pg.DB()
}

func MustExec(t *testing.T, db *bun.DB, query string, args ...any) {
	t.Helper()
	_, err := db.Exec(query, args...)
	require.NoError(t, err, fmt.Sprintf("failed to execute query: %s", query))
}

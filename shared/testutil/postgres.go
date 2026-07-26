package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
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
	sharedContainer        *PostgresContainer
	sharedRunningContainer *postgres.PostgresContainer
	sharedOnce             sync.Once
	sharedErr              error
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

// PostgresDSNEnv points the suite at an already-running Postgres instead of
// starting a throwaway container. Sharing one server across every `go test`
// process is what keeps the integration suite fast in CI.
const PostgresDSNEnv = "TRENOVA_TEST_POSTGRES_DSN"

// resolveAdminDSN returns a DSN for a Postgres server to run against, starting a
// container only when the caller has not supplied one.
func resolveAdminDSN(ctx context.Context, options PostgresOptions) (string, error) {
	if dsn := strings.TrimSpace(os.Getenv(PostgresDSNEnv)); dsn != "" {
		return dsn, nil
	}

	container, err := postgres.Run(ctx,
		options.Image,
		postgres.WithDatabase(options.Database),
		postgres.WithUsername(options.Username),
		postgres.WithPassword(options.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		return "", fmt.Errorf("failed to start postgres container: %w", err)
	}

	sharedRunningContainer = container

	host, err := container.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get postgres host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return "", fmt.Errorf("failed to get postgres port: %w", err)
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		options.Username, options.Password, host, port.Port(), options.Database), nil
}

// replaceDatabase swaps the database segment of a Postgres DSN.
func replaceDatabase(dsn, database string) string {
	base, query, hasQuery := strings.Cut(dsn, "?")
	slash := strings.LastIndex(base, "/")
	if slash < 0 {
		return dsn
	}

	replaced := base[:slash+1] + database
	if hasQuery {
		return replaced + "?" + query
	}

	return replaced
}

// createDatabaseWithRetry rides out the transient failures that show up when a
// dozen test processes create their databases on one server at the same time.
func createDatabaseWithRetry(ctx context.Context, adminDB *bun.DB, dbName string) error {
	stmt := fmt.Sprintf("CREATE DATABASE %s", dbName)

	var lastErr error
	for attempt := range 10 {
		if _, lastErr = adminDB.ExecContext(ctx, stmt); lastErr == nil {
			return nil
		}

		backoff := min(time.Duration(attempt+1)*250*time.Millisecond, 3*time.Second)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	return lastErr
}

func getSharedPostgres() (*PostgresContainer, error) {
	sharedOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		options := DefaultPostgresOptions()

		adminDSN, err := resolveAdminDSN(ctx, options)
		if err != nil {
			sharedErr = err
			return
		}

		// Many test processes share one server, so CREATE DATABASE can queue
		// behind other sessions; the default read timeout is too tight for that.
		adminSQL := sql.OpenDB(pgdriver.NewConnector(
			pgdriver.WithDSN(adminDSN),
			pgdriver.WithReadTimeout(2*time.Minute),
			pgdriver.WithWriteTimeout(2*time.Minute),
		))
		adminSQL.SetMaxOpenConns(2)
		adminDB := bun.NewDB(adminSQL, pgdialect.New())

		for i := range 30 {
			if pingErr := adminDB.PingContext(ctx); pingErr == nil {
				break
			}
			if i == 29 {
				adminDB.Close()
				sharedErr = fmt.Errorf("failed to connect to postgres after retries")
				return
			}
			time.Sleep(500 * time.Millisecond)
		}

		dbName := fmt.Sprintf("trenova_shared_test_%d", os.Getpid())

		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
		err = createDatabaseWithRetry(ctx, adminDB, dbName)
		adminDB.Close()
		if err != nil {
			sharedErr = fmt.Errorf("failed to create per-process database: %w", err)
			return
		}

		dsn := replaceDatabase(adminDSN, dbName)
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		if err := db.PingContext(ctx); err != nil {
			sharedErr = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		sharedContainer = &PostgresContainer{
			container: sharedRunningContainer,
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

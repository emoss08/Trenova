package seedtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/keel/migrate"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const (
	// PostgresDSNEnv points the integration suite at an already-running Postgres
	// (a CI service container, or the local docker-compose stack) instead of
	// starting a throwaway one. The server must have PostGIS available.
	PostgresDSNEnv = "TRENOVA_TEST_POSTGRES_DSN"

	PostgresImageEnv = "TRENOVA_TEST_POSTGRES_IMAGE"

	DefaultPostgresImage = "postgis/postgis:18-3.6-alpine"

	TemplateKey = "trenova-seedtest"

	postgresDatabase = "trenova_seed_test"
	postgresUser     = "test"
	postgresPassword = "test"
	databasePrefix   = "trenova_test_"

	templateBuildTimeout = 10 * time.Minute
	copyTimeout          = 90 * time.Second
	dropAttempts         = 3
)

var (
	sharedSeedEnv  *seedEnvironment
	sharedSeedOnce sync.Once
	sharedSeedErr  error
)

type seedEnvironment struct {
	adminDSN string
	set      migrate.Set
}

type TestDB struct {
	DB *bun.DB
	Tx bun.Tx
	t  *testing.T
}

func NewTestDB(t *testing.T, db *bun.DB) *TestDB {
	t.Helper()

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err, "failed to begin test transaction")

	return &TestDB{
		DB: db,
		Tx: tx,
		t:  t,
	}
}

func (tdb *TestDB) Rollback() {
	tdb.t.Helper()
	require.NoError(tdb.t, tdb.Tx.Rollback(), "failed to rollback test transaction")
}

func (tdb *TestDB) Commit() {
	tdb.t.Helper()
	require.NoError(tdb.t, tdb.Tx.Commit(), "failed to commit test transaction")
}

func (tdb *TestDB) Context() context.Context {
	return context.Background()
}

func chdirToServiceRoot(t *testing.T) {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	origDir := dir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "cmd", "cli")); err == nil {
				require.NoError(t, os.Chdir(dir))
				t.Cleanup(func() {
					os.Chdir(origDir) //nolint:errcheck
				})
				return
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatal("could not find TMS service root directory")
}

func openBunDB(dsn string, maxOpen int) (*bun.DB, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse test database DSN: %w", err)
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	sqldb := stdlib.OpenDB(*cfg)
	sqldb.SetMaxOpenConns(maxOpen)
	sqldb.SetMaxIdleConns(1)
	sqldb.SetConnMaxIdleTime(30 * time.Second)
	sqldb.SetConnMaxLifetime(2 * time.Minute)

	return bun.NewDB(sqldb, pgdialect.New()), nil
}

// resolveAdminDSN returns a DSN for a Postgres server to run the suite against.
//
// Prefer supplying PostgresDSNEnv: every `go test` process then shares one
// server, so the migrated template database is built once for the whole run
// instead of once per package. Without it each process starts its own
// container, which is correct but markedly slower.
func resolveAdminDSN(ctx context.Context) (string, error) {
	if dsn := strings.TrimSpace(os.Getenv(PostgresDSNEnv)); dsn != "" {
		return dsn, nil
	}

	req := testcontainers.ContainerRequest{
		Image:        PostgresImage(),
		ExposedPorts: []string{"5432/tcp"},
		Cmd:          []string{"postgres", "-c", "max_connections=300"},
		Env: map[string]string{
			"POSTGRES_DB":       postgresDatabase,
			"POSTGRES_USER":     postgresUser,
			"POSTGRES_PASSWORD": postgresPassword,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(1),
	}

	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to start postgis container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get postgis host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return "", fmt.Errorf("failed to get postgis port: %w", err)
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser,
		postgresPassword,
		host,
		port.Port(),
		postgresDatabase,
	), nil
}

func PostgresImage() string {
	if image := strings.TrimSpace(os.Getenv(PostgresImageEnv)); image != "" {
		return image
	}

	return DefaultPostgresImage
}

func waitForServer(ctx context.Context, dsn string) error {
	deadline := time.Now().Add(time.Minute)
	for {
		conn, err := pgx.Connect(ctx, dsn)
		if err == nil {
			return conn.Close(ctx)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for postgres to accept connections: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func templateOptions(env *seedEnvironment) migrate.TemplateOptions {
	return migrate.TemplateOptions{
		Admin:   env.adminDSN,
		Sources: env.set,
		Options: migrations.Options(),
		Key:     TemplateKey,
		Prefix:  databasePrefix,
	}
}

func getSharedSeedEnv() (*seedEnvironment, error) {
	sharedSeedOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), templateBuildTimeout)
		defer cancel()

		adminDSN, err := resolveAdminDSN(ctx)
		if err != nil {
			sharedSeedErr = err
			return
		}
		if err = waitForServer(ctx, adminDSN); err != nil {
			sharedSeedErr = err
			return
		}

		env := &seedEnvironment{adminDSN: adminDSN, set: migrations.Set()}
		dsn, drop, err := migrate.Template(ctx, templateOptions(env))
		if err != nil {
			sharedSeedErr = fmt.Errorf("failed to build the migrated template database: %w", err)
			return
		}
		if err = drop(ctx); err != nil {
			sharedSeedErr = fmt.Errorf("failed to drop the probe database %s: %w", dsn, err)
			return
		}

		sharedSeedEnv = env
	})

	return sharedSeedEnv, sharedSeedErr
}

func SetupTestDB(t *testing.T) (context.Context, *bun.DB, func()) {
	t.Helper()

	ctx, db, _, cleanup := SetupTestDBWithDSN(t)

	return ctx, db, cleanup
}

func SetupTestDBWithDSN(t *testing.T) (context.Context, *bun.DB, string, func()) {
	t.Helper()

	chdirToServiceRoot(t)

	env, err := getSharedSeedEnv()
	require.NoError(t, err, "failed to get shared seed environment")

	copyCtx, copyCancel := context.WithTimeout(context.Background(), copyTimeout)
	dsn, drop, err := migrate.Template(copyCtx, templateOptions(env))
	copyCancel()
	require.NoError(t, err, "failed to create isolated test database")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	db, err := openBunDB(dsn, 2)
	if err == nil {
		err = db.PingContext(ctx)
	}
	if err != nil {
		cancel()
		dropDatabase(t, drop, dsn)
		require.NoError(t, err, "failed to open isolated test database")
	}

	cleanup := func() {
		if closeErr := db.Close(); closeErr != nil {
			require.NoError(t, closeErr, "failed to close isolated test database")
		}
		dropDatabase(t, drop, dsn)
		cancel()
	}

	return ctx, db, dsn, cleanup
}

func dropDatabase(t *testing.T, drop func(context.Context) error, dsn string) {
	t.Helper()

	var err error
	for range dropAttempts {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = drop(dropCtx)
		dropCancel()
		if err == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Logf("failed to drop isolated test database %s: %v", redactDSN(dsn), err)
}

func redactDSN(dsn string) string {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return "<unparseable dsn>"
	}

	return cfg.Database
}

func BeginTx(t *testing.T, ctx context.Context, db *bun.DB) (context.Context, bun.Tx) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err, "failed to begin transaction")

	return ctx, tx
}

package seedtest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/migrations"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var (
	sharedSeedEnv     *seedEnvironment
	sharedSeedOnce    sync.Once
	sharedSeedErr     error
	sharedSeedCounter atomic.Uint64
)

type seedEnvironment struct {
	adminDB      *bun.DB
	adminDSN     string
	host         string
	port         string
	username     string
	password     string
	templateName string
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

func getSharedSeedEnv() (*seedEnvironment, error) {
	sharedSeedOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		req := testcontainers.ContainerRequest{
			Name:         "trenova-test-postgis",
			Image:        "postgis/postgis:16-3.4-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "trenova_seed_test",
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		}

		container, err := testcontainers.GenericContainer(
			ctx,
			testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
				Reuse:            true,
			},
		)
		if err != nil {
			sharedSeedErr = fmt.Errorf("failed to start postgis container: %w", err)
			return
		}

		host, err := container.Host(ctx)
		if err != nil {
			sharedSeedErr = fmt.Errorf("failed to get postgis host: %w", err)
			return
		}

		port, err := container.MappedPort(ctx, "5432")
		if err != nil {
			sharedSeedErr = fmt.Errorf("failed to get postgis port: %w", err)
			return
		}

		adminDSN := fmt.Sprintf(
			"postgres://test:test@%s:%s/trenova_seed_test?sslmode=disable",
			host,
			port.Port(),
		)
		adminSQL := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(adminDSN)))
		adminDB := bun.NewDB(adminSQL, pgdialect.New())

		for i := range 30 {
			if pingErr := adminDB.PingContext(ctx); pingErr == nil {
				break
			}
			if i == 29 {
				adminDB.Close()
				sharedSeedErr = fmt.Errorf("failed to connect to postgis container after retries")
				return
			}
			time.Sleep(500 * time.Millisecond)
		}

		dbName := fmt.Sprintf("trenova_seed_%d", os.Getpid())

		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		_, err = adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			sharedSeedErr = fmt.Errorf("failed to create per-process database: %w", err)
			return
		}

		dsn := fmt.Sprintf(
			"postgres://test:test@%s:%s/%s?sslmode=disable",
			host,
			port.Port(),
			dbName,
		)
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		if err := db.PingContext(ctx); err != nil {
			sharedSeedErr = fmt.Errorf("failed to ping postgis database: %w", err)
			return
		}

		if err := migrations.Run(ctx, db); err != nil {
			sharedSeedErr = fmt.Errorf("failed to run migrations: %w", err)
			return
		}

		if err := db.Close(); err != nil {
			sharedSeedErr = fmt.Errorf("failed to close template database connection: %w", err)
			return
		}

		sharedSeedEnv = &seedEnvironment{
			adminDB:      adminDB,
			adminDSN:     adminDSN,
			host:         host,
			port:         port.Port(),
			username:     "test",
			password:     "test",
			templateName: dbName,
		}
	})

	return sharedSeedEnv, sharedSeedErr
}

func SetupTestDB(t *testing.T) (context.Context, *bun.DB, func()) {
	t.Helper()

	chdirToServiceRoot(t)

	env, err := getSharedSeedEnv()
	require.NoError(t, err, "failed to get shared seed environment")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	dbName := fmt.Sprintf("trenova_test_%d_%d", os.Getpid(), sharedSeedCounter.Add(1))
	_, err = env.adminDB.ExecContext(
		ctx,
		fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", dbName, env.templateName),
	)
	require.NoError(t, err, "failed to create isolated test database")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		env.username,
		env.password,
		env.host,
		env.port,
		dbName,
	)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	require.NoError(t, db.PingContext(ctx), "failed to ping isolated test database")

	cleanup := func() {
		require.NoError(t, db.Close(), "failed to close isolated test database")
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dropCancel()

		_, _ = env.adminDB.ExecContext(
			dropCtx,
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()",
			dbName,
		)
		_, dropErr := env.adminDB.ExecContext(
			dropCtx,
			fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName),
		)
		require.NoError(t, dropErr, "failed to drop isolated test database")
		cancel()
	}

	return ctx, db, cleanup
}

func BeginTx(t *testing.T, ctx context.Context, db *bun.DB) (context.Context, bun.Tx) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err, "failed to begin transaction")

	return ctx, tx
}

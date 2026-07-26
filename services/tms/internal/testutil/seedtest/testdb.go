package seedtest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

const (
	// PostgresDSNEnv points the integration suite at an already-running Postgres
	// (a CI service container, or the local docker-compose stack) instead of
	// starting a throwaway one. The server must have PostGIS available.
	PostgresDSNEnv = "TRENOVA_TEST_POSTGRES_DSN"

	postgresImage    = "postgis/postgis:16-3.4-alpine"
	postgresDatabase = "trenova_seed_test"
	postgresUser     = "test"
	postgresPassword = "test"

	// templateLockKey serialises template-database construction across the test
	// processes that run in parallel, so migrations execute exactly once.
	templateLockKey int64 = 8021990

	templateBuildTimeout = 10 * time.Minute
	createFromTemplate   = 90 * time.Second
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

func openBunDB(dsn string, maxOpen int) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	sqldb.SetMaxOpenConns(maxOpen)
	sqldb.SetMaxIdleConns(1)
	sqldb.SetConnMaxIdleTime(30 * time.Second)
	sqldb.SetConnMaxLifetime(2 * time.Minute)

	return bun.NewDB(sqldb, pgdialect.New())
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
		Image:        postgresImage,
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

func waitForDB(ctx context.Context, db *bun.DB) error {
	deadline := time.Now().Add(time.Minute)
	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("timed out waiting for postgres to accept connections")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func databaseExists(ctx context.Context, adminDB *bun.DB, name string) (bool, error) {
	var exists bool
	err := adminDB.QueryRowContext(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)",
		name,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to look up database %q: %w", name, err)
	}

	return exists, nil
}

// ensureTemplateDatabase builds the migrated template exactly once per server,
// even when many test binaries race to create it. Migrations run into a staging
// database that is renamed into place only on success, so the presence of the
// template is proof that it is fully migrated.
func ensureTemplateDatabase(
	ctx context.Context,
	adminDB *bun.DB,
	adminDSN string,
	templateName string,
) error {
	conn, err := adminDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire advisory lock connection: %w", err)
	}
	defer conn.Close()

	if _, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock(?)", templateLockKey); err != nil {
		return fmt.Errorf("failed to acquire template advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock(?)", templateLockKey)
	}()

	exists, err := databaseExists(ctx, adminDB, templateName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	stagingName := templateName + "_building"
	if _, err = adminDB.ExecContext(
		ctx,
		fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", stagingName),
	); err != nil {
		return fmt.Errorf("failed to drop stale staging database: %w", err)
	}
	if _, err = adminDB.ExecContext(
		ctx,
		fmt.Sprintf("CREATE DATABASE %s", stagingName),
	); err != nil {
		return fmt.Errorf("failed to create staging database: %w", err)
	}

	stagingDB := openBunDB(replaceDatabase(adminDSN, stagingName), 2)
	if err = stagingDB.PingContext(ctx); err != nil {
		stagingDB.Close() //nolint:errcheck // best effort on the failure path
		return fmt.Errorf("failed to connect to staging database: %w", err)
	}
	if err = migrations.Run(ctx, stagingDB); err != nil {
		stagingDB.Close() //nolint:errcheck // best effort on the failure path
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	if err = stagingDB.Close(); err != nil {
		return fmt.Errorf("failed to close staging database connection: %w", err)
	}

	if _, err = adminDB.ExecContext(
		ctx,
		fmt.Sprintf("ALTER DATABASE %s RENAME TO %s", stagingName, templateName),
	); err != nil {
		return fmt.Errorf("failed to publish template database: %w", err)
	}

	return nil
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

func getSharedSeedEnv() (*seedEnvironment, error) {
	sharedSeedOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), templateBuildTimeout)
		defer cancel()

		adminDSN, err := resolveAdminDSN(ctx)
		if err != nil {
			sharedSeedErr = err
			return
		}

		adminDB := openBunDB(adminDSN, 4)
		if err = waitForDB(ctx, adminDB); err != nil {
			adminDB.Close() //nolint:errcheck // best effort on the failure path
			sharedSeedErr = err
			return
		}

		fingerprint, err := migrations.Fingerprint()
		if err != nil {
			adminDB.Close() //nolint:errcheck // best effort on the failure path
			sharedSeedErr = err
			return
		}
		templateName := "trenova_tpl_" + fingerprint[:16]

		if err = ensureTemplateDatabase(ctx, adminDB, adminDSN, templateName); err != nil {
			adminDB.Close() //nolint:errcheck // best effort on the failure path
			sharedSeedErr = err
			return
		}

		sharedSeedEnv = &seedEnvironment{
			adminDB:      adminDB,
			adminDSN:     adminDSN,
			templateName: templateName,
		}
	})

	return sharedSeedEnv, sharedSeedErr
}

// createDatabaseFromTemplate retries while other processes hold a session on the
// template; Postgres refuses CREATE DATABASE ... TEMPLATE in that window.
func createDatabaseFromTemplate(
	ctx context.Context,
	adminDB *bun.DB,
	dbName, templateName string,
) error {
	stmt := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", dbName, templateName)
	deadline := time.Now().Add(createFromTemplate)

	var lastErr error
	for attempt := 0; ; attempt++ {
		_, lastErr = adminDB.ExecContext(ctx, stmt)
		if lastErr == nil {
			return nil
		}
		if !strings.Contains(lastErr.Error(), "being accessed by other users") {
			return lastErr
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out cloning template database: %w", lastErr)
		}

		backoff := min(time.Duration(attempt+1)*50*time.Millisecond, time.Second)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
}

func SetupTestDB(t *testing.T) (context.Context, *bun.DB, func()) {
	t.Helper()

	chdirToServiceRoot(t)

	env, err := getSharedSeedEnv()
	require.NoError(t, err, "failed to get shared seed environment")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	dbName := fmt.Sprintf("trenova_test_%d_%d", os.Getpid(), sharedSeedCounter.Add(1))
	cloneCtx, cloneCancel := context.WithTimeout(context.Background(), createFromTemplate)
	err = createDatabaseFromTemplate(cloneCtx, env.adminDB, dbName, env.templateName)
	cloneCancel()
	if err != nil {
		cancel()
		require.NoError(t, err, "failed to create isolated test database")
	}

	db := openBunDB(replaceDatabase(env.adminDSN, dbName), 2)
	if pingErr := db.PingContext(ctx); pingErr != nil {
		cancel()
		require.NoError(t, pingErr, "failed to ping isolated test database")
	}

	cleanup := func() {
		if closeErr := db.Close(); closeErr != nil {
			require.NoError(t, closeErr, "failed to close isolated test database")
		}

		var dropErr error
		for range 3 {
			dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, _ = env.adminDB.ExecContext(
				dropCtx,
				fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName),
			)
			dropCancel()

			dropCtx, dropCancel = context.WithTimeout(context.Background(), 10*time.Second)
			_, dropErr = env.adminDB.ExecContext(
				dropCtx,
				fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName),
			)
			dropCancel()
			if dropErr == nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if dropErr != nil {
			t.Logf("failed to drop isolated test database %s: %v", dbName, dropErr)
		}
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

package migrator_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/migrator"
	sqlitemigrations "github.com/emoss08/trenova/internal/infrastructure/sqlite/migrations"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/migrate"

	_ "modernc.org/sqlite"
)

func TestSQLiteMigrationsApplyCleanly(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	migrator := migrate.NewMigrator(db, sqlitemigrations.Migrations)
	require.NoError(t, migrator.Init(ctx))

	group, err := migrator.Migrate(ctx)
	require.NoError(t, err, "the generated SQLite migration set must apply without error")
	require.NotZero(t, group.ID, "expected at least one migration group to be applied")

	var tables int
	require.NoError(t, db.NewRaw(
		"SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'",
	).Scan(ctx, &tables))

	require.Greater(t, tables, 200, "expected the bulk of the schema to be created")
}

func TestSQLiteMigrationsAreIdempotent(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	migrator := migrate.NewMigrator(db, sqlitemigrations.Migrations)
	require.NoError(t, migrator.Init(ctx))

	_, err := migrator.Migrate(ctx)
	require.NoError(t, err)

	group, err := migrator.Migrate(ctx)
	require.NoError(t, err)
	require.Zero(t, group.ID, "a second migrate run must be a no-op")
}

func TestSQLiteCoreTablesExist(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	migrator := migrate.NewMigrator(db, sqlitemigrations.Migrations)
	require.NoError(t, migrator.Init(ctx))
	_, err := migrator.Migrate(ctx)
	require.NoError(t, err)

	for _, table := range []string{
		"business_units", "organizations", "users", "shipments", "shipment_moves",
		"stops", "workers", "customers", "locations", "tractors", "trailers",
	} {
		var count int
		require.NoError(t, db.NewRaw(
			"SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table,
		).Scan(ctx, &count))

		require.Equalf(t, 1, count, "expected table %q to exist on SQLite", table)
	}
}

func TestSQLiteResetRecreatesSchema(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	m := migrator.NewMigrator(&common.DatabaseConfig{
		DB:          db,
		Environment: common.EnvDevelopment,
	})
	require.NoError(t, m.Initialize(ctx))

	_, err := m.Migrate(ctx, common.OperationOptions{Force: true})
	require.NoError(t, err)

	result, err := m.Reset(ctx, common.OperationOptions{Force: true})
	require.NoError(t, err, "reset must work on SQLite, not just PostgreSQL")
	require.True(t, result.Success)

	var applied int
	require.NoError(t, db.NewRaw("SELECT count(*) FROM bun_migrations").Scan(ctx, &applied))
	require.Positive(t, applied, "reset must leave the migration history populated")

	var tables int
	require.NoError(t, db.NewRaw(
		"SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'",
	).Scan(ctx, &tables))
	require.Greater(t, tables, 200, "reset must rebuild the schema")
}

func newSQLiteDB(t *testing.T) *bun.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "trenova-test.db")

	sqldb, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(0)&_txlock=immediate")
	require.NoError(t, err)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

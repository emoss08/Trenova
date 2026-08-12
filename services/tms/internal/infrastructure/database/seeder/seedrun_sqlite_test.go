package seeder_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	sqlitemigrations "github.com/emoss08/trenova/internal/infrastructure/sqlite/migrations"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/migrate"

	_ "modernc.org/sqlite"
)

// Migrating and then seeding a throwaway SQLite database is the only thing that
// catches dialect problems in the seeds themselves, as opposed to the schema.
func TestFullSeedRunOnSQLite(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "seedrun.db")

	// Seeds resolve their YAML data files relative to the module root.
	t.Chdir("../../../..")

	sqldb, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(0)&_txlock=immediate")
	require.NoError(t, err)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	db.RegisterModel(domainregistry.RegisterManyToManyEntities()...)
	db.RegisterModel(domainregistry.RegisterEntities()...)

	migrator := migrate.NewMigrator(db, sqlitemigrations.Migrations)
	require.NoError(t, migrator.Init(ctx))
	_, err = migrator.Migrate(ctx)
	require.NoError(t, err)

	cfg := &config.Config{}
	cfg.Database.Driver = "sqlite"
	cfg.App.Env = "development"
	cfg.System.SystemUserPassword = "seed-test-password"

	registry := seeder.NewRegistry()
	seeds.Register(registry)

	tracker := seeder.NewTracker(db)
	require.NoError(t, tracker.Initialize(ctx))

	engine := seeder.NewEngine(db, registry, cfg)
	engine.SetTracker(tracker)

	_, err = engine.Execute(ctx, seeder.ExecuteOptions{Environment: "development"})
	require.NoError(t, err)
}

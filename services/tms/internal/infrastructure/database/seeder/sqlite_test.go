package seeder

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	_ "modernc.org/sqlite"
)

type stubSeed struct {
	name    string
	version string
}

func (s *stubSeed) Name() string                       { return s.name }
func (s *stubSeed) Version() string                    { return s.version }
func (s *stubSeed) Description() string                { return "stub seed" }
func (s *stubSeed) Dependencies() []string             { return nil }
func (s *stubSeed) Environments() []common.Environment { return nil }
func (s *stubSeed) Run(context.Context, bun.Tx) error  { return nil }
func (s *stubSeed) Down(context.Context, bun.Tx) error { return nil }
func (s *stubSeed) CanRollback() bool                  { return true }

func TestTrackerInitializeOnSQLite(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	tracker := NewTracker(db)
	require.NoError(t, tracker.Initialize(ctx), "seed tracking DDL must be valid SQLite")

	require.NoError(t, tracker.Initialize(ctx), "Initialize must be idempotent")

	requireTableExists(t, db, "seed_history")
}

func TestTrackerRecordsAndReadsOnSQLite(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	tracker := NewTracker(db)
	require.NoError(t, tracker.Initialize(ctx))

	seed := &stubSeed{name: "TestSeed", version: "1.0.0"}
	env := common.EnvDevelopment

	applied, err := tracker.IsApplied(ctx, seed, env)
	require.NoError(t, err)
	require.False(t, applied)

	require.NoError(t, tracker.RecordSuccess(ctx, seed, env, 25*time.Millisecond))

	applied, err = tracker.IsApplied(ctx, seed, env)
	require.NoError(t, err)
	require.True(t, applied, "a recorded seed must read back as applied")

	require.NoError(
		t,
		tracker.RecordSuccess(ctx, seed, env, 30*time.Millisecond),
		"re-recording must upsert rather than violate the unique constraint",
	)

	statuses, err := tracker.GetStatus(ctx)
	require.NoError(t, err)
	require.Len(t, statuses, 1)
}

func TestTrackerRecordFailureKeepsAppliedSeedOnSQLite(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	tracker := NewTracker(db)
	require.NoError(t, tracker.Initialize(ctx))

	seed := &stubSeed{name: "TestSeed", version: "1.0.0"}
	env := common.EnvDevelopment

	require.NoError(t, tracker.RecordSuccess(ctx, seed, env, time.Millisecond))
	require.NoError(t, tracker.RecordFailure(ctx, seed, env, errors.New("boom")))

	applied, err := tracker.IsApplied(ctx, seed, env)
	require.NoError(t, err)
	require.True(
		t,
		applied,
		"a failed re-run must not demote an already applied seed",
	)
}

func TestTrackerRecordRollbackOnSQLite(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	tracker := NewTracker(db)
	require.NoError(t, tracker.Initialize(ctx))

	seed := &stubSeed{name: "TestSeed", version: "1.0.0"}
	env := common.EnvDevelopment

	require.NoError(t, tracker.RecordSuccess(ctx, seed, env, time.Millisecond))
	require.NoError(t, tracker.RecordRollback(ctx, seed, env))

	applied, err := tracker.IsApplied(ctx, seed, env)
	require.NoError(t, err)
	require.False(t, applied, "a rolled back seed must be re-appliable")
}

func TestRollbackTrackerOnSQLite(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	tracker := NewRollbackTracker(db)
	require.NoError(t, tracker.Initialize(ctx), "rollback DDL must be valid SQLite")
	require.NoError(t, tracker.Initialize(ctx), "Initialize must be idempotent")

	requireTableExists(t, db, "seed_rollbacks")

	require.NoError(t, tracker.RecordSuccess(
		ctx, "TestSeed", "1.0.0", common.EnvDevelopment, 3, time.Second,
	))
	require.NoError(t, tracker.RecordFailure(
		ctx, "TestSeed", "1.0.0", common.EnvDevelopment, "boom",
	))

	var count int
	require.NoError(t, db.NewRaw("SELECT count(*) FROM seed_rollbacks").Scan(ctx, &count))
	require.Equal(t, 2, count)
}

func TestEverySupportedDialectHasSchemaFiles(t *testing.T) {
	for _, kind := range []dbdialect.Kind{dbdialect.Postgres, dbdialect.SQLite} {
		for _, name := range []string{seedHistorySchema, seedRollbacksSchema} {
			statements, err := loadSchema(kind, name)
			require.NoErrorf(t, err, "%s is missing schema %q", kind, name)
			require.NotEmptyf(t, statements, "%s schema %q is empty", kind, name)
		}
	}
}

func TestLoadSchemaRejectsUnknownName(t *testing.T) {
	_, err := loadSchema(dbdialect.SQLite, "not_a_table")
	require.Error(t, err)
}

func requireTableExists(t *testing.T, db *bun.DB, table string) {
	t.Helper()

	var count int
	require.NoError(t, db.NewRaw(
		"SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table,
	).Scan(t.Context(), &count))

	require.Equalf(t, 1, count, "expected table %q to exist", table)
}

func newSQLiteDB(t *testing.T) *bun.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "seeder-test.db")

	sqldb, err := sql.Open("sqlite", "file:"+path+"?_txlock=immediate")
	require.NoError(t, err)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

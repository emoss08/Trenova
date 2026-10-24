//go:build integration

package bunadopt_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel/migrate"
	"github.com/emoss08/trenova/internal/infrastructure/database/bunadopt"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bunEraVersion = "20261231019000"

func connect(t *testing.T) (context.Context, *pgx.Conn) {
	t.Helper()

	ctx, _, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(ctx)) })

	return ctx, conn
}

func exec(t *testing.T, ctx context.Context, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()

	_, err := conn.Exec(ctx, sql, args...)
	require.NoError(t, err)
}

func bunEra(t *testing.T, ctx context.Context, conn *pgx.Conn, skip ...string) {
	t.Helper()

	exec(t, ctx, conn, `DROP TABLE keel_migrations`)
	exec(t, ctx, conn, `CREATE TABLE bun_migrations (
		id bigserial PRIMARY KEY,
		name varchar,
		group_id bigint,
		migrated_at timestamptz NOT NULL DEFAULT current_timestamp
	)`)
	exec(
		t,
		ctx,
		conn,
		`CREATE TABLE bun_migration_locks (id bigserial PRIMARY KEY, table_name varchar)`,
	)

	ms, err := migrations.Set().Migrations()
	require.NoError(t, err)
	skipped := make(map[string]bool, len(skip))
	for _, v := range skip {
		skipped[v] = true
	}
	for _, m := range ms {
		if m.Version <= bunEraVersion && !skipped[m.Version] {
			exec(t, ctx, conn,
				`INSERT INTO bun_migrations (name, group_id) VALUES ($1, 1)`, m.Version)
		}
	}
}

func count(t *testing.T, ctx context.Context, conn *pgx.Conn, sql string) int {
	t.Helper()

	var n int
	require.NoError(t, conn.QueryRow(ctx, sql).Scan(&n))

	return n
}

func TestAdoptBaselinesWhatBunAppliedAndUpRunsTheRest(t *testing.T) {
	ctx, conn := connect(t)
	bunEra(t, ctx, conn)

	set := migrations.Set()
	result, err := bunadopt.Adopt(ctx, conn, set, migrations.Options())
	require.NoError(t, err)
	assert.True(t, result.Adopted())
	assert.Equal(t, bunEraVersion, result.Version)

	applied := count(t, ctx, conn, `SELECT count(*) FROM bun_migrations`)
	assert.Equal(t, applied, result.Recorded)
	assert.Equal(t, applied, count(t, ctx, conn, `SELECT count(*) FROM keel_migrations`))

	m, err := migrate.New(conn, set, migrations.Options())
	require.NoError(t, err)
	res, err := m.Up(ctx, migrate.UpOptions{})
	require.NoError(t, err)

	ms, err := set.Migrations()
	require.NoError(t, err)
	assert.Len(t, res.Applied, len(ms)-applied, "only the migrations newer than Bun's run")
	assert.Zero(t, count(t, ctx, conn,
		`SELECT count(*) FROM pg_class WHERE relname IN ('bun_migrations', 'bun_migration_locks')`),
		"the keel bookkeeping migration drops Bun's tables")

	again, err := bunadopt.Adopt(ctx, conn, set, migrations.Options())
	require.NoError(t, err)
	assert.False(t, again.Adopted())
}

func TestAdoptRefusesADatabaseBunLeftAGapIn(t *testing.T) {
	ctx, conn := connect(t)

	ms, err := migrations.Set().Migrations()
	require.NoError(t, err)
	gap := ms[len(ms)/2]
	bunEra(t, ctx, conn, gap.Version)

	_, err = bunadopt.Adopt(ctx, conn, migrations.Set(), migrations.Options())
	require.ErrorIs(t, err, bunadopt.ErrUnadoptable)
	assert.Contains(t, err.Error(), gap.ID())
	assert.Zero(t, count(t, ctx, conn,
		`SELECT count(*) FROM pg_class WHERE relname = 'keel_migrations'`),
		"a refused adoption records nothing")
}

func TestAdoptLeavesAKeelDatabaseAlone(t *testing.T) {
	ctx, conn := connect(t)

	before := count(t, ctx, conn, `SELECT count(*) FROM keel_migrations`)
	result, err := bunadopt.Adopt(ctx, conn, migrations.Set(), migrations.Options())
	require.NoError(t, err)
	assert.False(t, result.Adopted())
	assert.Equal(t, before, count(t, ctx, conn, `SELECT count(*) FROM keel_migrations`))
}

func TestAdoptIgnoresAnEmptyBunTable(t *testing.T) {
	ctx, conn := connect(t)
	exec(t, ctx, conn, `TRUNCATE keel_migrations`)
	exec(t, ctx, conn, `CREATE TABLE bun_migrations (id bigserial PRIMARY KEY, name varchar)`)

	result, err := bunadopt.Adopt(ctx, conn, migrations.Set(), migrations.Options())
	require.NoError(t, err)
	assert.False(t, result.Adopted())
	assert.Zero(t, count(t, ctx, conn, `SELECT count(*) FROM keel_migrations`))
}

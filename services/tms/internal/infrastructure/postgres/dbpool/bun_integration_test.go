//go:build integration

package dbpool_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type typeProbe struct {
	bun.BaseModel `bun:"table:dbpool_type_probes"`

	ID         int64             `bun:"id,pk"`
	Labels     []string          `bun:"labels,array"`
	Counts     []int64           `bun:"counts,array"`
	Statuses   []string          `bun:"statuses,array"`
	Status     string            `bun:"status"`
	Attributes map[string]any    `bun:"attributes,type:jsonb"`
	Headers    map[string]string `bun:"headers,type:jsonb"`
	Amount     decimal.Decimal   `bun:"amount,type:numeric(12,2)"`
	OccurredAt time.Time         `bun:"occurred_at"`
	LocalAt    time.Time         `bun:"local_at"`
	Day        time.Time         `bun:"day"`
	Payload    []byte            `bun:"payload"`
	Active     bool              `bun:"active"`
	Note       *string           `bun:"note"`
	Ratio      float64           `bun:"ratio"`
}

const typeProbeTable = `CREATE TABLE dbpool_type_probes (
	id bigint PRIMARY KEY,
	labels text[] NOT NULL,
	counts bigint[] NOT NULL,
	statuses status_enum[] NOT NULL,
	status status_enum NOT NULL,
	attributes jsonb NOT NULL,
	headers jsonb NOT NULL,
	amount numeric(12, 2) NOT NULL,
	occurred_at timestamptz NOT NULL,
	local_at timestamp NOT NULL,
	day date NOT NULL,
	payload bytea NOT NULL,
	active boolean NOT NULL,
	note text,
	ratio double precision NOT NULL
)`

func openPool(t *testing.T) dbpool.Pool {
	t.Helper()

	_, _, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	pool, err := dbpool.Open(t.Context(), &dbpool.Options{DSN: dsn, MaxConns: 4})
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Bun.Close() })

	return pool
}

func TestBunRoundTripsColumnTypesOverPgx(t *testing.T) {
	pool := openPool(t)
	ctx := t.Context()

	_, err := pool.Bun.ExecContext(ctx, typeProbeTable)
	require.NoError(t, err)

	want := &typeProbe{
		ID:         1,
		Labels:     []string{"a", "b,c", `quote"d`, ""},
		Counts:     []int64{1, -2, 9_007_199_254_740_993},
		Statuses:   []string{"Active", "Inactive"},
		Status:     "Active",
		Attributes: map[string]any{"nested": map[string]any{"n": float64(2)}, "list": []any{"x"}},
		Headers:    map[string]string{"k": "v"},
		Amount:     decimal.RequireFromString("12345.67"),
		OccurredAt: time.Date(2026, 3, 8, 2, 30, 15, 123456000, time.UTC),
		LocalAt:    time.Date(2026, 3, 8, 2, 30, 15, 0, time.UTC),
		Day:        time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC),
		Payload:    []byte{0x00, 0xff, 0x10},
		Active:     true,
		Ratio:      0.125,
	}
	_, err = pool.Bun.NewInsert().Model(want).Exec(ctx)
	require.NoError(t, err)

	got := new(typeProbe)
	require.NoError(t, pool.Bun.NewSelect().Model(got).Where("id = ?", 1).Scan(ctx))

	assert.Equal(t, want.Labels, got.Labels)
	assert.Equal(t, want.Counts, got.Counts)
	assert.Equal(t, want.Statuses, got.Statuses)
	assert.Equal(t, want.Status, got.Status)
	assert.Equal(t, want.Attributes, got.Attributes)
	assert.Equal(t, want.Headers, got.Headers)
	assert.True(t, want.Amount.Equal(got.Amount), got.Amount.String())
	assert.True(t, want.OccurredAt.Equal(got.OccurredAt), got.OccurredAt.String())
	assert.True(t, want.LocalAt.Equal(got.LocalAt), got.LocalAt.String())
	assert.True(t, want.Day.Equal(got.Day), got.Day.String())
	assert.Equal(t, want.Payload, got.Payload)
	assert.Equal(t, want.Active, got.Active)
	assert.Nil(t, got.Note)
	assert.InDelta(t, want.Ratio, got.Ratio, 0)
}

func TestBunScansIntoScalarsOverPgx(t *testing.T) {
	pool := openPool(t)
	ctx := t.Context()

	var (
		count    int64
		labels   []string
		raw      map[string]any
		at       time.Time
		nullable *string
	)
	err := pool.Bun.NewRaw(
		"SELECT count(*), ARRAY['x', 'y'], '{\"a\": [1, true]}'::jsonb, now(), NULL::text FROM pg_class",
	).Scan(ctx, &count, pgdialect.Array(&labels), &raw, &at, &nullable)
	require.NoError(t, err)
	assert.Positive(t, count)
	assert.Equal(t, []string{"x", "y"}, labels)
	assert.Equal(t, map[string]any{"a": []any{float64(1), true}}, raw)
	assert.WithinDuration(t, time.Now(), at, time.Minute)
	assert.Nil(t, nullable)
}

func TestBunRunsMultiStatementQueriesOverPgx(t *testing.T) {
	pool := openPool(t)
	ctx := t.Context()

	_, err := pool.Bun.ExecContext(ctx, "CREATE TABLE dbpool_multi (id int); INSERT INTO dbpool_multi VALUES (1), (2)")
	require.NoError(t, err)

	var ids []int
	require.NoError(t, pool.Bun.NewRaw("SELECT id FROM dbpool_multi ORDER BY id").Scan(ctx, &ids))
	assert.Equal(t, []int{1, 2}, ids)
}

func TestBunAndKeelShareOnePool(t *testing.T) {
	pool := openPool(t)

	assert.Same(t, pool.Keel, dbpool.KeelOf(pool.Bun.DB))

	conn, err := pool.Bun.Conn(t.Context())
	require.NoError(t, err)
	defer conn.Close()

	pgxConn, err := dbpool.PgxConn(conn)
	require.NoError(t, err)
	require.NotNil(t, pgxConn)

	var backend int
	require.NoError(t, conn.QueryRowContext(t.Context(), "SELECT pg_backend_pid()").Scan(&backend))

	var viaPgx int
	require.NoError(t, pgxConn.QueryRow(context.Background(), "SELECT pg_backend_pid()").Scan(&viaPgx))
	assert.Equal(t, backend, viaPgx, "the pgx connection is the one under Bun's driver connection")

	stat := pool.Keel.Pool().Stat()
	assert.Equal(t, int32(1), stat.AcquiredConns(), "Bun's dedicated connection is a pgx pool connection")
}

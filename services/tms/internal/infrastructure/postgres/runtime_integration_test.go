//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/querybudget"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/internal/testutil/keeltest/keeltestdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func cachedFleetCodes(ctx context.Context, conn *Connection) ([]string, error) {
	return dbtx.Read(ctx, conn, func(ctx context.Context) ([]string, error) {
		s := keeltestdb.FleetCodes
		rows, err := s.Query().Order(s.Code.Asc()).Limit(20).Cache().List(ctx, conn.Keel(ctx))
		if err != nil {
			return nil, err
		}

		codes := make([]string, len(rows))
		for i, row := range rows {
			codes[i] = row.Code
		}

		return codes, nil
	})
}

func transactionIsolation(
	ctx context.Context,
	conn *Connection,
	opts kernelports.TxOptions,
) (string, error) {
	var level string
	err := conn.Transact(ctx, opts, func(ctx context.Context) error {
		var err error
		level, err = keel.QueryValue[string](ctx, conn.Keel(ctx), "SHOW transaction_isolation")
		return err
	})

	return level, err
}

func TestRuntime_ReadsRunAtRepeatableReadUnlessTheyAskOtherwise(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	level, err := transactionIsolation(ctx, conn, kernelports.TxOptions{ReadOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "repeatable read", level)

	level, err = transactionIsolation(ctx, conn, kernelports.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelReadCommitted,
	})
	require.NoError(t, err)
	assert.Equal(t, "read committed", level)

	level, err = transactionIsolation(ctx, conn, kernelports.TxOptions{})
	require.NoError(t, err)
	assert.Equal(t, "read committed", level, "writes keep the server default")

	level, err = transactionIsolation(
		ctx,
		conn,
		kernelports.TxOptions{Isolation: sql.LevelSerializable},
	)
	require.NoError(t, err)
	assert.Equal(t, "serializable", level)
}

func TestRuntime_ADbtxReadSeesOneSnapshot(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	before, after, err := dbtx.Read2(
		ctx,
		conn,
		func(ctx context.Context) ([]string, []string, error) {
			first, err := keelCodes(ctx, conn.Keel(ctx))
			if err != nil {
				return nil, nil, err
			}

			if err = rawInsertFleetCode(
				dbscope.WithTenant(t.Context(), f.tenantA),
				WrapDB(f.admin),
				f.tenantA,
				"LATER",
			); err != nil {
				return nil, nil, err
			}

			second, err := keelCodes(ctx, conn.Keel(ctx))
			return first, second, err
		},
	)
	require.NoError(t, err)
	assert.Equal(t, before, after, "a commit made during the read stays invisible to it")
	assert.Contains(t, f.codes(t, f.tenantA), "LATER")
}

func TestRuntime_TheResultCacheServesARepeatedReadPerTenantAndReadScope(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	f.cfg.Database.Cache = config.DatabaseCacheConfig{Enabled: true, LocalBytes: 8 << 20}
	conn := f.connect(t)

	tenantA := dbscope.WithTenant(t.Context(), f.tenantA)
	first, err := cachedFleetCodes(tenantA, conn)
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA"}, first)

	stats := conn.tenantKeel.CacheStats()
	require.True(t, stats.Listening, "the listener connects before the pool serves reads")
	assert.Equal(t, int64(1), stats.Misses)

	second, err := cachedFleetCodes(tenantA, conn)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Equal(t, int64(1), conn.tenantKeel.CacheStats().Hits)

	tenantB, err := cachedFleetCodes(dbscope.WithTenant(t.Context(), f.tenantB), conn)
	require.NoError(t, err)
	assert.Equal(t, []string{"FCB"}, tenantB, "another tenant never reads tenant A's entry")

	narrowed := readscope.With(tenantA, func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{
			"worker": {Scope: readscope.ScopeTeam, Team: []pulid.ID{pulid.MustNew("usr_")}},
		}, nil
	})
	_, err = cachedFleetCodes(narrowed, conn)
	require.NoError(t, err)
	assert.Equal(
		t,
		int64(1),
		conn.tenantKeel.CacheStats().Hits,
		"a narrower read scope is its own entry",
	)
	assert.Equal(t, int64(3), conn.tenantKeel.CacheStats().Misses)

	_, err = dbtx.Write(tenantA, conn, func(ctx context.Context) (*keeltest.FleetCode, error) {
		row := keelFleetCode(f.tenantA, "NEW")
		return row, keeltestdb.FleetCodes.Create(ctx, conn.Keel(ctx), row)
	})
	require.NoError(t, err)

	afterWrite, err := cachedFleetCodes(tenantA, conn)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{"FCA", "NEW"},
		afterWrite,
		"this process's write invalidates its entries at once",
	)
}

func TestRuntime_ConnectionReportsTenantConcurrencyAndCacheStats(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	f.cfg.Database.MaxOpenConns = 8
	conn := f.connect(t)

	assert.Equal(t, 2, conn.settings.tenantConcurrency, "a quarter of the pool by default")

	stats := conn.runtimeStats()
	assert.Zero(t, stats.Cache, "the cache is off in this configuration")
	assert.Zero(t, stats.TenantWaits)
}

func TestRuntime_ABudgetLogsAStatementRepeatedPastItsLimitAndLetsTheWorkFinish(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)

	core, logs := observer.New(zapcore.WarnLevel)
	cfg := &config.Config{}
	cfg.Database.Budget.MaxRepeats = 2
	budgets := querybudget.New(querybudget.Params{Config: cfg, Logger: zap.New(core)})

	ctx := budgets.Start(dbscope.WithTenant(t.Context(), f.tenantA), zap.String("route", "/test"))
	for range 4 {
		codes, err := fleetCodes(ctx, conn)
		require.NoError(t, err, "a budget past its limit only reports")
		assert.Equal(t, []string{"FCA"}, codes)
	}

	report, ok := keel.BudgetReportFrom(ctx)
	require.True(t, ok)
	require.NotEmpty(t, report.Repeats)
	assert.Equal(t, "SELECT code FROM fleet_codes ORDER BY code", report.Repeats[0].SQL)
	assert.Equal(t, 4, report.Repeats[0].Count)

	entries := logs.All()
	require.Len(t, entries, 1, "a budget reports once")
	assert.Equal(t, "/test", entries[0].ContextMap()["route"])
}

func TestRuntime_AHardBudgetFailsTheReadPastIt(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)

	ctx := keel.Budget(dbscope.WithTenant(t.Context(), f.tenantA), keel.MaxRepeats(1))
	_, err := fleetCodes(ctx, conn)
	require.NoError(t, err)

	_, err = fleetCodes(ctx, conn)
	require.ErrorIs(t, err, keel.ErrBudgetExceeded)
}

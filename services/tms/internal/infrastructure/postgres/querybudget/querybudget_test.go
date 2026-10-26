package querybudget

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newBudgets(t *testing.T, maxRepeats int) (*Budgets, *observer.ObservedLogs) {
	t.Helper()

	core, logs := observer.New(zapcore.WarnLevel)
	cfg := &config.Config{}
	cfg.Database.Budget.MaxRepeats = maxRepeats

	return New(Params{Config: cfg, Logger: zap.New(core)}), logs
}

func TestNewDefaultsToTenRepeats(t *testing.T) {
	t.Parallel()

	budgets, _ := newBudgets(t, 0)
	assert.Equal(t, 10, budgets.maxRepeats)
}

func TestStartPutsABudgetOnTheContext(t *testing.T) {
	t.Parallel()

	budgets, _ := newBudgets(t, 3)

	_, ok := keel.BudgetReportFrom(t.Context())
	require.False(t, ok)

	report, ok := keel.BudgetReportFrom(budgets.Start(t.Context()))
	require.True(t, ok)
	assert.Zero(t, report.RoundTrips)

	var nilBudgets *Budgets
	_, ok = keel.BudgetReportFrom(nilBudgets.Start(t.Context()))
	assert.False(t, ok, "without budgets the context is unchanged")
}

func TestExceededLogsTheRepeatsPastTheLimitWithTheirCallSites(t *testing.T) {
	t.Parallel()

	budgets, logs := newBudgets(t, 3)
	budgets.exceeded(t.Context(), keel.BudgetReport{
		RoundTrips: 14,
		Statements: 15,
		Repeats: []keel.Repeat{
			{
				SQL:       "SELECT id FROM stops WHERE shipment_move_id = $1",
				Count:     12,
				CallSites: []string{"stoprepository/stop.go:88", "shipmentservice/get.go:41"},
			},
			{SQL: "SELECT id FROM customers WHERE id = $1", Count: 2},
		},
	}, []zap.Field{zap.String("route", "/api/v1/shipments/:id/")})

	entries := logs.All()
	require.Len(t, entries, 1)
	fields := entries[0].ContextMap()
	assert.Equal(t, "/api/v1/shipments/:id/", fields["route"])
	assert.Equal(t, int64(3), fields["max_repeats"])
	assert.Equal(t, int64(14), fields["round_trips"])
	assert.Equal(t, int64(15), fields["statements"])

	repeated, ok := fields["repeats"].([]any)
	require.True(t, ok)
	require.Len(t, repeated, 1, "a text within the limit is not reported")
	first, ok := repeated[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "SELECT id FROM stops WHERE shipment_move_id = $1", first["sql"])
	assert.Equal(t, 12, first["count"])
	assert.Equal(
		t,
		[]any{"stoprepository/stop.go:88", "shipmentservice/get.go:41"},
		first["call_sites"],
	)
}

func TestOverBudgetKeepsTheMostRepeatedFew(t *testing.T) {
	t.Parallel()

	report := keel.BudgetReport{}
	for i := range 8 {
		report.Repeats = append(report.Repeats, keel.Repeat{SQL: "SELECT $1", Count: 20 - i})
	}

	kept := overBudget(report, 10)
	require.Len(t, kept, maxLoggedRepeats)
	assert.Equal(t, 20, kept[0].Count)
}

func TestTruncateBoundsLoggedStatements(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "SELECT", truncate("SELECT", 10))
	assert.Equal(t, "SELE…", truncate("SELECT", 4))
}

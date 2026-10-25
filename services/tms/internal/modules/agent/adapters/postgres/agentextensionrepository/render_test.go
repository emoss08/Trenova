package agentextensionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension/agentextensiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReserveIncrementsOnlyBelowTheLimit(t *testing.T) {
	t.Parallel()

	src, conflict, cols := reserve(&agentextensionports.ReserveExtensionRequestParams{
		Type:  agentextension.TypeExa,
		Day:   20260923,
		Limit: 250,
		Now:   1_790_000_000,
	})
	sql, args, err := agentextensiondb.ExtensionUsageDailies.UpsertFromSQL(
		keeltest.TenantContext(t),
		keeltest.RenderDB(t),
		src,
		conflict,
		cols...,
	)
	require.NoError(t, err)

	assert.Contains(t, sql,
		`ON CONFLICT ("organization_id", "business_unit_id", "extension_type", "day") DO UPDATE SET`)
	assert.Contains(t, sql, `"requests" = aeud."requests" + $`)
	assert.Contains(t, sql, `"updated_at" = EXCLUDED."updated_at"`)
	assert.Contains(t, sql, `WHERE aeud."requests" < $`)
	assert.Contains(t, args, "Exa")
	assert.Contains(t, args, int64(250))
}

func TestRecordOutcomeAddsToTheTenantsDay(t *testing.T) {
	t.Parallel()

	q, sets := recordOutcome(agentextensionports.RecordExtensionOutcomeParams{
		Type:    agentextension.TypeExa,
		Day:     20260923,
		Failed:  true,
		CostUSD: decimal.RequireFromString("0.005"),
		Now:     1_790_000_000,
	})
	sql, args := keeltest.Render(t, q.Limit(1))

	assert.Contains(t, sql, `aeud."organization_id" = $1 AND aeud."business_unit_id" = $2`)
	assert.Contains(t, sql, `aeud."extension_type" = 'Exa' AND aeud."day" = $3`)
	assert.Equal(t, []any{"org_1", "bu_1", int64(20260923)}, args)
	assert.Len(t, sets, 3)
}

func TestSummarizeReadsTheMonthForOneTenant(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, summarize(agentextensionports.SummarizeExtensionUsageParams{
		FromDay: 20260901,
		Today:   20260923,
	}))

	assert.Contains(t, sql, `FILTER (WHERE aeud."day" = $`)
	assert.Contains(t, sql, `aeud."day" >= $`)
	assert.Contains(t, sql, `aeud."day" <= $`)
	assert.Contains(t, sql, `GROUP BY aeud."extension_type"`)
	assert.Contains(t, args, int64(20260901))
	assert.Contains(t, args, int64(20260923))
	assert.Contains(t, args, "org_1")
}

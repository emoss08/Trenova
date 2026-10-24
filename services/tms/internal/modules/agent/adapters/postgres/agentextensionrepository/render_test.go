package agentextensionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestReserveIncrementsOnlyBelowTheLimit(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	args := reserveArgs(agentextensionports.ReserveExtensionRequestParams{
		TenantInfo: tenant,
		Type:       agentextension.TypeExa,
		Day:        20260923,
		Limit:      250,
		Now:        1_790_000_000,
	})

	assert.Contains(t, reserveSQL,
		"ON CONFLICT (organization_id, business_unit_id, extension_type, day) DO UPDATE")
	assert.Contains(t, reserveSQL, "requests = agent_extension_usage_daily.requests + 1")
	assert.Contains(t, reserveSQL, "updated_at = EXCLUDED.updated_at")
	assert.Contains(t, reserveSQL, "WHERE agent_extension_usage_daily.requests < $6")
	assert.Equal(t, []any{
		tenant.OrgID.String(),
		tenant.BuID.String(),
		"Exa",
		20260923,
		int64(1_790_000_000),
		250,
	}, args)
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

	assert.Contains(t, sql, `agextu."business_unit_id" = $1 AND agextu."organization_id" = $2`)
	assert.Contains(t, sql, `agextu."extension_type" = $3 AND agextu."day" = $4`)
	assert.Equal(t, []any{"bu_1", "org_1", "Exa", int64(20260923)}, args)
	assert.Len(t, sets, 3)
}

func TestSummarizeReadsTheMonthForOneTenant(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, summarize(agentextensionports.SummarizeExtensionUsageParams{
		FromDay: 20260901,
		Today:   20260923,
	}))

	assert.Contains(t, sql, `FILTER (WHERE agextu."day" = $`)
	assert.Contains(t, sql, `agextu."day" >= $`)
	assert.Contains(t, sql, `agextu."day" <= $`)
	assert.Contains(t, sql, `GROUP BY agextu."extension_type"`)
	assert.Contains(t, args, int64(20260901))
	assert.Contains(t, args, int64(20260923))
	assert.Contains(t, args, "org_1")
}

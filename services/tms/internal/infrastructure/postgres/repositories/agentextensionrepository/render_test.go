package agentextensionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func renderDB() *bun.DB {
	return bun.NewDB(nil, pgdialect.New())
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestReserveIncrementsOnlyBelowTheLimit(t *testing.T) {
	t.Parallel()

	sql := buildReserve(renderDB(), repositories.ReserveExtensionRequestParams{
		TenantInfo: testTenant(),
		Type:       agentextension.TypeExa,
		Day:        20260923,
		Limit:      250,
		Now:        1_790_000_000,
	}).String()

	assert.Contains(t, sql,
		"ON CONFLICT (organization_id, business_unit_id, extension_type, day) DO UPDATE", sql)
	assert.Contains(t, sql, "requests = agextu.requests + 1", sql)
	assert.Contains(t, sql, "updated_at = EXCLUDED.updated_at", sql)
	assert.Contains(t, sql, "WHERE (agextu.requests < 250)", sql)
	assert.Contains(t, sql, "'Exa'", sql)
	assert.Contains(t, sql, "20260923", sql)
}

func TestRecordOutcomeAddsToTheTenantsDay(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	sql := buildRecordOutcome(renderDB(), repositories.RecordExtensionOutcomeParams{
		TenantInfo: tenant,
		Type:       agentextension.TypeExa,
		Day:        20260923,
		Failed:     true,
		CostUSD:    decimal.RequireFromString("0.005"),
		Now:        1_790_000_000,
	}).String()

	assert.Contains(t, sql, "failures = failures + 1", sql)
	assert.Contains(t, sql, "cost_usd = cost_usd + '0.005'", sql)
	assert.Contains(t, sql, tenant.OrgID.String(), sql)
	assert.Contains(t, sql, tenant.BuID.String(), sql)
	assert.Contains(t, sql, "extension_type = 'Exa'", sql)
}

func TestRecordOutcomeNeverSubtractsCost(t *testing.T) {
	t.Parallel()

	sql := buildRecordOutcome(renderDB(), repositories.RecordExtensionOutcomeParams{
		TenantInfo: testTenant(),
		Type:       agentextension.TypeExa,
		Day:        20260923,
		CostUSD:    decimal.NewFromInt(-3),
	}).String()

	assert.Contains(t, sql, "failures = failures + 0", sql)
	assert.Contains(t, sql, "cost_usd = cost_usd + '0'", sql)
}

func TestSummarizeReadsTheMonthForOneTenant(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	sql := buildSummarize(renderDB(), repositories.SummarizeExtensionUsageParams{
		TenantInfo: tenant,
		FromDay:    20260901,
		Today:      20260923,
	}).String()

	assert.Contains(t, sql, "FILTER (WHERE agextu.day = 20260923)", sql)
	assert.Contains(t, sql, "agextu.day >= 20260901", sql)
	assert.Contains(t, sql, "agextu.day <= 20260923", sql)
	assert.Contains(t, sql, tenant.OrgID.String(), sql)
	assert.Contains(t, sql, `GROUP BY "agextu"."extension_type"`, sql)
}

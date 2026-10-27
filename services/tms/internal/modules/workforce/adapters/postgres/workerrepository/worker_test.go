package workerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestSelectOptionsQuery_OwnerOperatorsMatchTheTypeOrTheirCurrentPayProfile(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, selectOptionsQuery(true, 1_700_000_000).Limit(10))

	assert.Contains(t, sql, `w."type" = 'Contractor'`)
	assert.Contains(t, sql, `AND wpa."worker_id" = w."id"`)
	assert.Contains(t, sql, `AND wpa."organization_id" = w."organization_id"`)
	assert.Contains(t, sql, `AND wpa."business_unit_id" = w."business_unit_id"`)
	assert.Contains(t, sql, `LEFT JOIN "fleet_codes" fc ON `)
	assert.Contains(t, sql, `dpp."classification" = 'OwnerOperator'`)
	assert.Contains(t, args, int64(1_700_000_000))
}

func TestSelectOptionsQuery_LeavesThePayProfileOutByDefault(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, selectOptionsQuery(false, 1_700_000_000).Limit(10))

	assert.NotContains(t, sql, "worker_pay_assignments")
}

func TestWorkerQuery_LoadsOnlyTheRequestedRelations(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, workerQuery(&workerports.GetWorkerByIDRequest{
		ID:                  "wrk_1",
		IncludeProfile:      true,
		IncludeLicenseState: true,
		IncludeManager:      true,
	}))

	assert.Contains(t, sql, `"worker_profiles"`)
	assert.Contains(t, sql, `"us_states"`)
	assert.Contains(t, sql, `"users"`)
	assert.NotContains(t, sql, `"fleet_codes"`)
}

func TestRosterAttentionQuery_CountsEveryTileInOnePass(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, rosterAttentionQuery(1_800_000_000))

	assert.Contains(t, sql, `FROM "workers" w JOIN "worker_profiles" wp ON `)
	assert.Contains(t, sql, `count(*) FILTER (WHERE wp."compliance_status" = 'NonCompliant')`)
	assert.Contains(t, sql, `wp."next_credential_expiry" IS NOT NULL`)
	assert.Contains(t, args, int64(1_800_000_000))
	assert.Equal(t, []any{"bu_1", "org_1", int64(1_800_000_000)}, args)
}

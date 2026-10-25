package briefingrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing/briefingdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestForDayReadsTheSharedBriefingWhenThereIsNoReader(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, forDay(briefing.RoleKey("dispatch"), "2026-01-01", nil))

	assert.Contains(t, sql, `"user_id" IS NULL`, sql)
	assert.Contains(t, args, "2026-01-01")
}

func TestForDayReadsOnePersonsBriefing(t *testing.T) {
	t.Parallel()

	userID := pulid.ID("usr_1")
	sql, args := keeltest.Render(t, forDay(briefing.RoleKey("dispatch"), "2026-01-01", &userID))

	assert.Contains(t, sql, `"user_id" = $`, sql)
	assert.NotContains(t, sql, "IS NULL", sql)
	assert.Contains(t, args, userID.String())
}

func TestListCapsTheLimitAndOrdersNewestDayFirst(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, listQuery(&briefingports.ListBriefingsQuery{Limit: 500}))

	assert.Contains(t, sql, `"briefing_date" DESC`, sql)
	assert.Contains(t, sql, "LIMIT 90", sql)
	assert.NotContains(t, sql, `"role_key"`+" = ", sql)
}

func TestStaleBatchSelectsTheOldestDaysAcrossTenants(t *testing.T) {
	t.Parallel()

	b := briefingdb.Briefings
	sql, args := keeltest.RenderIn(
		t,
		dbscope.WithSystem(t.Context(), "test"),
		b.AllTenants().Where(staleBatch("2026-01-01", 500)),
	)

	assert.Contains(
		t,
		sql,
		`."id" IN (SELECT stale."id" FROM "assistant_briefings" stale WHERE stale."briefing_date" < $1 ORDER BY stale."briefing_date" ASC LIMIT 500)`,
		sql,
	)
	assert.Equal(t, []any{"2026-01-01"}, args)
}

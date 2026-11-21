package briefingrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing/briefingdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures/managedfeaturesdb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type entitledScopes struct{ entitlementports.Filter }

func (entitledScopes) ForWork(_ context.Context, organization keel.Col) (keel.Pred, error) {
	s := managedfeaturesdb.States
	return organization.InQuery(s.Pick(s.ScopeKey)), nil
}

type noWork struct{ entitlementports.Filter }

func (noWork) ForWork(context.Context, keel.Col) (keel.Pred, error) {
	return nil, entitlement.ErrNoWork
}

func TestStaleBatchSelectsTheOldestDaysOfTheEntitledTenants(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithSystem(t.Context(), "test")
	repo := &repository{entitlements: entitledScopes{}}
	batch, err := repo.staleBatch(ctx, "2026-01-01", 500)
	require.NoError(t, err)

	b := briefingdb.Briefings
	sql, args := keeltest.RenderIn(t, ctx, b.AllTenants().Where(batch))

	assert.Contains(
		t,
		sql,
		`."id" IN (SELECT stale."id" FROM "assistant_briefings" stale `+
			`WHERE stale."organization_id" IN (SELECT mfs."scope_key" FROM "managed_feature_states" mfs) `+
			`AND stale."briefing_date" < $1 ORDER BY stale."briefing_date" ASC LIMIT 500)`,
		sql,
	)
	assert.Equal(t, []any{"2026-01-01"}, args)
}

func TestStaleBatchRefusesASweepWithNoWork(t *testing.T) {
	t.Parallel()

	repo := &repository{entitlements: noWork{}}
	_, err := repo.staleBatch(dbscope.WithSystem(t.Context(), "test"), "2026-01-01", 500)

	require.ErrorIs(t, err, entitlement.ErrNoWork)
}

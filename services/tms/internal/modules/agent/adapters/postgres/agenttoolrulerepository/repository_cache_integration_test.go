//go:build integration

package agenttoolrulerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolRuleOverridesAreCachedPerTenantUntilAnOverrideChanges(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "TR")
	cached := cachetest.Open(t, ctx, dsn)
	repo := New(Params{DB: postgres.NewTestConnection(cached)})
	actx := tenantA.Ctx(ctx)

	var overrides []*agenttooltrust.ToolRuleOverride
	readFor := func(data *seedtest.TestData) func() error {
		return func() error {
			var readErr error
			overrides, readErr = repo.List(data.Ctx(ctx), data.Info())
			return readErr
		}
	}

	t.Run("an organization without overrides is cached as empty", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(tenantA))
		assert.Empty(t, overrides)
		cachetest.RequireHit(t, cached, readFor(tenantA))
		assert.Empty(t, overrides)
	})

	created, err := repo.Create(actx, &agenttooltrust.ToolRuleOverride{
		OrganizationID: tenantA.Organization.ID,
		BusinessUnitID: tenantA.BusinessUnit.ID,
		ToolName:       "shipment_update",
		MaxTier:        agent.TierPropose,
		Reason:         "hold shipment edits for review",
	})
	require.NoError(t, err)

	t.Run("a new override invalidates the tenant's rules", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(tenantA))
		require.Len(t, overrides, 1)
		assert.Equal(t, agent.TierPropose, overrides[0].MaxTier)
		cachetest.RequireHit(t, cached, readFor(tenantA))
		require.Len(t, overrides, 1)
	})

	t.Run("another tenant never reads the first tenant's rules", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(tenantB))
		assert.Empty(t, overrides)
	})

	t.Run("changing the override invalidates the rules", func(t *testing.T) {
		created.MaxTier = agent.TierActWithApproval
		_, err := repo.Update(actx, created)
		require.NoError(t, err)

		cachetest.RequireMiss(t, cached, readFor(tenantA))
		require.Len(t, overrides, 1)
		assert.Equal(t, agent.TierActWithApproval, overrides[0].MaxTier)
	})
}

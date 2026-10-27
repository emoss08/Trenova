//go:build integration

package iamrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEnabledAccessPoliciesAreCachedPerTenantUntilAPolicyChanges(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "AP")
	cached := cachetest.Open(t, ctx, dsn)
	repo := New(Params{DB: postgres.NewTestConnection(cached), Logger: zap.NewNop()})
	actx := tenantA.Ctx(ctx)

	var policies []*iam.AccessPolicy
	readFor := func(data *seedtest.TestData) func() error {
		return func() error {
			var readErr error
			policies, readErr = repo.ListEnabledTenantAccessPolicies(
				data.Ctx(ctx),
				iamports.IAMTenantPolicyLookupRequest{
					OrganizationID: data.Organization.ID,
					BusinessUnitID: data.BusinessUnit.ID,
				},
			)
			return readErr
		}
	}

	t.Run("an empty policy set is cached", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(tenantA))
		assert.Empty(t, policies)
		cachetest.RequireHit(t, cached, readFor(tenantA))
		assert.Empty(t, policies)
	})

	created, err := repo.CreateAccessPolicy(actx, &iam.AccessPolicy{
		OrganizationID: tenantA.Organization.ID,
		BusinessUnitID: tenantA.BusinessUnit.ID,
		Name:           "Deny shipment reads",
		Resource:       permission.ResourceShipment.String(),
		Operation:      string(permission.OpRead),
		Effect:         iam.PolicyEffectDeny,
		Priority:       10,
		Enabled:        true,
	})
	require.NoError(t, err)

	t.Run("a new policy invalidates the tenant's set", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(tenantA))
		require.Len(t, policies, 1)
		assert.Equal(t, created.ID, policies[0].ID)
		cachetest.RequireHit(t, cached, readFor(tenantA))
		require.Len(t, policies, 1)
	})

	t.Run("another tenant never reads the first tenant's policies", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(tenantB))
		assert.Empty(t, policies)
	})

	t.Run("disabling the policy invalidates the set", func(t *testing.T) {
		created.Enabled = false
		_, err := repo.UpdateAccessPolicy(actx, created)
		require.NoError(t, err)

		cachetest.RequireMiss(t, cached, readFor(tenantA))
		assert.Empty(t, policies)
	})
}

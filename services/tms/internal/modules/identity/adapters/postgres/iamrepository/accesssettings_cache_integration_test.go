//go:build integration

package iamrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAccessSettingsAreCachedPerTenantIncludingTheDefault(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "AS")
	cached := cachetest.Open(t, ctx, dsn)
	repo := NewAccessSettingsRepository(Params{
		DB:     postgres.NewTestConnection(cached),
		Logger: zap.NewNop(),
	})

	var settings *iam.AccessSettings
	readFor := func(data *seedtest.TestData) func() error {
		return func() error {
			var readErr error
			settings, readErr = repo.Get(data.Ctx(ctx), data.Info())
			return readErr
		}
	}

	cachetest.RequireMiss(t, cached, readFor(tenantA))
	cachetest.RequireHit(t, cached, readFor(tenantA))
	assert.False(
		t,
		settings.RequireRoleActivation,
		"an organization without a row reads the default",
	)
	assert.Equal(t, tenantA.Organization.ID, settings.OrganizationID)

	cachetest.RequireMiss(t, cached, readFor(tenantB))

	_, err := repo.Save(tenantA.Ctx(ctx), &iam.AccessSettings{
		OrganizationID:        tenantA.Organization.ID,
		BusinessUnitID:        tenantA.BusinessUnit.ID,
		RequireRoleActivation: true,
	})
	require.NoError(t, err)

	cachetest.RequireMiss(t, cached, readFor(tenantA))
	assert.True(t, settings.RequireRoleActivation, "the saved row replaces the cached default")
	cachetest.RequireHit(t, cached, readFor(tenantA))
	assert.True(t, settings.RequireRoleActivation)

	cachetest.RequireHit(t, cached, readFor(tenantB))
	assert.False(t, settings.RequireRoleActivation, "another tenant's entry is its own")
	assert.Equal(t, tenantB.Organization.ID, settings.OrganizationID)
}

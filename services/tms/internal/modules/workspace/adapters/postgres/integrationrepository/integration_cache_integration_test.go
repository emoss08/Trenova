//go:build integration

package integrationrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestIntegrationByTypeIsCachedIncludingItsAbsence(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "IN")
	cached := cachetest.Open(t, ctx, dsn)
	repo := New(Params{DB: postgres.NewTestConnection(cached), Logger: zap.NewNop()})
	actx := tenantA.Ctx(ctx)

	var found *integration.Integration
	var readErr error
	readFor := func(data *seedtest.TestData) func() error {
		return func() error {
			found, readErr = repo.GetByType(data.Ctx(ctx), data.Info(), integration.TypePCMiler)
			if errortypes.IsNotFoundError(readErr) {
				return nil
			}
			return readErr
		}
	}

	t.Run("a missing integration is cached as not found", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(tenantA))
		assert.True(t, errortypes.IsNotFoundError(readErr), "got %v", readErr)
		cachetest.RequireHit(t, cached, readFor(tenantA))
		assert.True(t, errortypes.IsNotFoundError(readErr), "got %v", readErr)
		assert.Nil(t, found)
	})

	t.Run("configuring it invalidates the absence", func(t *testing.T) {
		_, err := repo.Upsert(actx, &integration.Integration{
			OrganizationID: tenantA.Organization.ID,
			BusinessUnitID: tenantA.BusinessUnit.ID,
			Type:           integration.TypePCMiler,
			Name:           "PC*Miler",
			Category:       integration.CategoryMappingRouting,
			Enabled:        true,
			Configuration:  map[string]any{"baseUrl": "https://pcmiler.example.com"},
		})
		require.NoError(t, err)

		cachetest.RequireMiss(t, cached, readFor(tenantA))
		require.NoError(t, readErr)
		require.NotNil(t, found)
		assert.True(t, found.Enabled)
		cachetest.RequireHit(t, cached, readFor(tenantA))
		require.NotNil(t, found)
		assert.Equal(t, "https://pcmiler.example.com", found.Configuration["baseUrl"])
	})

	t.Run("another tenant keeps its own answer", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(tenantB))
		assert.True(t, errortypes.IsNotFoundError(readErr), "got %v", readErr)
	})
}

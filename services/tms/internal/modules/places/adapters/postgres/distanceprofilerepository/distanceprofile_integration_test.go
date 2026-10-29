//go:build integration

package distanceprofilerepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile/distanceprofiledb"
	"github.com/emoss08/trenova/internal/modules/places/ports/distanceprofileports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedDistanceProfile(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	name string,
) *distanceprofile.DistanceProfile {
	t.Helper()

	entity := distanceprofile.NewDefault(tenant.Organization.ID, tenant.BusinessUnit.ID)
	entity.Name = name
	entity.IsDefault = false
	require.NoError(t, distanceprofiledb.DistanceProfiles.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetByIDsReturnsOnlyTheTenantsDistanceProfiles(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "DP")
	first := seedDistanceProfile(t, ctx, db, tenantA, "Shortest")
	second := seedDistanceProfile(t, ctx, db, tenantA, "Fastest")
	foreign := seedDistanceProfile(t, ctx, db, tenantB, "Foreign")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		distanceprofileports.GetDistanceProfilesByIDsRequest{
			TenantInfo:         tenantA.Info(),
			DistanceProfileIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("dp_")},
		},
	)
	require.NoError(t, err)

	names := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		names[entity.ID] = entity.Name
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "Shortest", second.ID: "Fastest"}, names)

	none, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		distanceprofileports.GetDistanceProfilesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}

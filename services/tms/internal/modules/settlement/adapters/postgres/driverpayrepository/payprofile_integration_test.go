//go:build integration

package driverpayrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedPayProfile(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	name string,
) *driverpay.PayProfile {
	t.Helper()

	entity := &driverpay.PayProfile{Name: name}
	require.NoError(t, driverpaydb.PayProfiles.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetByIDsReturnsOnlyTheTenantsPayProfiles(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "PP")
	first := seedPayProfile(t, ctx, db, tenantA, "Linehaul")
	second := seedPayProfile(t, ctx, db, tenantA, "Local")
	foreign := seedPayProfile(t, ctx, db, tenantB, "Foreign")

	repo := NewPayProfile(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	profiles, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		driverpayports.GetPayProfilesByIDsRequest{
			TenantInfo: tenantA.Info(),
			ProfileIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("dpp_")},
		},
	)
	require.NoError(t, err)

	names := make(map[pulid.ID]string, len(profiles))
	for _, profile := range profiles {
		names[profile.ID] = profile.Name
		assert.Empty(t, profile.Components, "a batch read loads no components")
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "Linehaul", second.ID: "Local"}, names)

	none, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		driverpayports.GetPayProfilesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}

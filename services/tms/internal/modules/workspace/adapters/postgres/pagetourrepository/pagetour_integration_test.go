//go:build integration

package pagetourrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/pagetourrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagetour"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/pagetourports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpsert_KeepsOneViewATourAndOnlyTheCallersOwn(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	u := tenantdb.Users
	users, err := u.AllTenants().
		Where(u.CurrentOrganizationID.IsNotNull()).
		Order(u.ID.Asc()).
		Limit(2).
		List(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	require.Len(t, users, 2, "the seed has at least two people")
	ana := pagination.TenantInfo{
		OrgID: users[0].CurrentOrganizationID, BuID: users[0].BusinessUnitID, UserID: users[0].ID,
	}
	ben := pagination.TenantInfo{
		OrgID: users[1].CurrentOrganizationID, BuID: users[1].BusinessUnitID, UserID: users[1].ID,
	}
	require.Equal(t, ana.OrgID, ben.OrgID, "both people work in the seeded organization")
	ctx = dbscope.WithTenant(ctx, ana.DBTenant())
	repo := pagetourrepository.New(pagetourrepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	})
	view := func(
		who pagination.TenantInfo,
		tour string,
		version int32,
		outcome pagetour.Outcome,
	) *pagetour.View {
		return &pagetour.View{
			OrganizationID: who.OrgID,
			BusinessUnitID: who.BuID,
			UserID:         who.UserID,
			Tour:           tour,
			TourVersion:    version,
			Outcome:        outcome,
			SeenAt:         1_800_000_000 + int64(version),
		}
	}

	seen, err := repo.Exists(
		ctx,
		&pagetourports.ViewRequest{TenantInfo: ana, Tour: "freight-outlook"},
	)
	require.NoError(t, err)
	assert.False(t, seen)

	first, err := repo.Upsert(ctx, view(ana, "freight-outlook", 1, pagetour.OutcomeSkipped))
	require.NoError(t, err)
	assert.False(t, first.ID.IsNil())
	again, err := repo.Upsert(ctx, view(ana, "freight-outlook", 2, pagetour.OutcomeFinished))
	require.NoError(t, err)
	assert.Equal(t, first.ID, again.ID, "seeing a tour again replaces the earlier view")
	_, err = repo.Upsert(ctx, view(ana, "console", 1, pagetour.OutcomeFinished))
	require.NoError(t, err)
	_, err = repo.Upsert(ctx, view(ben, "freight-outlook", 1, pagetour.OutcomeSkipped))
	require.NoError(t, err)

	views, err := repo.List(ctx, ana)
	require.NoError(t, err)
	require.Len(t, views, 2, "one row a tour, and nobody else's")
	assert.Equal(t, "console", views[0].Tour, "listed by tour")
	outlook := views[1]
	assert.Equal(t, "freight-outlook", outlook.Tour)
	assert.Equal(t, int32(2), outlook.TourVersion)
	assert.Equal(t, pagetour.OutcomeFinished, outlook.Outcome)
	assert.Equal(t, int64(1_800_000_002), outlook.SeenAt)
	assert.Equal(t, ana.UserID, outlook.UserID)

	count, err := repo.CountForUser(ctx, ana)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	seen, err = repo.Exists(
		ctx,
		&pagetourports.ViewRequest{TenantInfo: ana, Tour: "freight-outlook"},
	)
	require.NoError(t, err)
	assert.True(t, seen)

	theirs, err := repo.List(ctx, ben)
	require.NoError(t, err)
	require.Len(t, theirs, 1)
	assert.Equal(t, pagetour.OutcomeSkipped, theirs[0].Outcome)
}

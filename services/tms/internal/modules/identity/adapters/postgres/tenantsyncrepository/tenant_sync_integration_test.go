//go:build integration

package tenantsyncrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/entitledrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/managedfeaturesrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestListTenantsPageOnASharedServerSkipsTenantsOutsideTheNeed(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	withAI := seedtest.SeedAdditionalTenant(t, ctx, db, "EA")
	withoutAI := seedtest.SeedAdditionalTenant(t, ctx, db, "EB")
	suspended := seedtest.SeedAdditionalTenant(t, ctx, db, "EC")
	unplanned := seedtest.SeedAdditionalTenant(t, ctx, db, "ED")

	conn := postgres.WrapDB(db)
	states := managedfeaturesrepository.New(
		managedfeaturesrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	plan := func(tenant *seedtest.TestData, features ...platformcatalog.FeatureKey) managedscope.Scope {
		t.Helper()
		scope := managedscope.Organization(dbscope.Tenant{
			OrganizationID: tenant.Organization.ID,
			BusinessUnitID: tenant.BusinessUnit.ID,
		})
		_, err := states.Replace(ctx, &managedfeaturesports.ReplaceRequest{
			Scope:    scope,
			Received: &managedfeatures.Received{Features: features},
			Now:      100,
		})
		require.NoError(t, err)
		return scope
	}
	plan(withAI, platformcatalog.FeatureAIDesk)
	plan(withoutAI)
	_, err := states.SetLifecycle(ctx, &managedfeaturesports.LifecycleRequest{
		Scope:     plan(suspended, platformcatalog.FeatureAIDesk),
		Lifecycle: managedfeatures.LifecycleSuspended,
	})
	require.NoError(t, err)

	repo := New(Params{
		DB: conn,
		Entitlements: entitledrepository.New(entitledrepository.Params{
			DB:     conn,
			Scopes: managedscope.NewResolver(true),
			Gate:   plantest.AllFeatures(),
		}),
	})
	listed := func(need entitlement.Need) map[pulid.ID]bool {
		t.Helper()
		found := map[pulid.ID]bool{}
		after := pulid.Nil
		for {
			page, pageErr := repo.ListTenantsPage(ctx, need, after, 2)
			require.NoError(t, pageErr)
			for _, tenant := range page {
				found[tenant.OrgID] = true
			}
			if len(page) < 2 {
				return found
			}
			after = page[len(page)-1].OrgID
		}
	}

	ai := listed(entitlement.For(platformcatalog.FeatureAIDesk))
	assert.True(t, ai[withAI.Organization.ID])
	assert.False(t, ai[withoutAI.Organization.ID], "a plan without the add-on")
	assert.False(t, ai[suspended.Organization.ID], "a suspended tenant")
	assert.False(t, ai[unplanned.Organization.ID], "a tenant the control plane sent no plan for")

	active := listed(entitlement.AnyActive())
	assert.True(t, active[withAI.Organization.ID])
	assert.True(t, active[withoutAI.Organization.ID])
	assert.False(t, active[suspended.Organization.ID])
	assert.False(t, active[unplanned.Organization.ID])

	_, err = repo.ListTenantsPage(ctx, entitlement.Need{}, pulid.Nil, 10)
	require.ErrorIs(t, err, entitlement.ErrNoNeed)
}

func TestListTenantsPageOnADedicatedInstanceListsEveryOrganization(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	first := seedtest.SeedAdditionalTenant(t, ctx, db, "FA")
	second := seedtest.SeedAdditionalTenant(t, ctx, db, "FB")
	conn := postgres.WrapDB(db)
	repo := New(Params{
		DB: conn,
		Entitlements: entitledrepository.New(entitledrepository.Params{
			DB:     conn,
			Scopes: managedscope.NewResolver(false),
			Gate:   plantest.AllFeatures(),
		}),
	})

	page, err := repo.ListTenantsPage(
		ctx,
		entitlement.For(platformcatalog.FeatureAIDesk),
		pulid.Nil,
		1000,
	)
	require.NoError(t, err)
	ids := map[pulid.ID]bool{}
	for _, tenant := range page {
		ids[tenant.OrgID] = true
	}
	assert.True(t, ids[first.Organization.ID])
	assert.True(t, ids[second.Organization.ID])
}

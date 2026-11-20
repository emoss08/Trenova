//go:build integration

package entitledrepository

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/managedfeaturesrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures/managedfeaturesdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type sharedServer struct {
	ctx       context.Context
	db        *keel.DB
	filter    entitlementports.Filter
	withAI    dbscope.Tenant
	withoutAI dbscope.Tenant
	suspended dbscope.Tenant
	noPlan    dbscope.Tenant
}

func newTenant() dbscope.Tenant {
	return dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func setupSharedServer(t *testing.T) *sharedServer {
	t.Helper()
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.WrapDB(db)
	states := managedfeaturesrepository.New(
		managedfeaturesrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	s := &sharedServer{
		ctx: ctx,
		db:  db,
		filter: New(Params{
			DB:     conn,
			Scopes: managedscope.NewResolver(true),
			Gate:   plantest.AllFeatures(),
		}),
		withAI:    newTenant(),
		withoutAI: newTenant(),
		suspended: newTenant(),
		noPlan:    newTenant(),
	}
	plan := func(tenant dbscope.Tenant, features ...platformcatalog.FeatureKey) {
		t.Helper()
		_, err := states.Replace(ctx, &managedfeaturesports.ReplaceRequest{
			Scope:    managedscope.Organization(tenant),
			Received: &managedfeatures.Received{Features: features},
			Now:      100,
		})
		require.NoError(t, err)
	}
	plan(s.withAI, platformcatalog.FeatureAIDesk)
	plan(s.withoutAI, platformcatalog.FeatureAccounting)
	plan(s.suspended, platformcatalog.FeatureAIDesk)
	_, err := states.SetLifecycle(ctx, &managedfeaturesports.LifecycleRequest{
		Scope:     managedscope.Organization(s.suspended),
		Lifecycle: managedfeatures.LifecycleSuspended,
	})
	require.NoError(t, err)
	_, err = states.SetLifecycle(ctx, &managedfeaturesports.LifecycleRequest{
		Scope:     managedscope.Organization(s.noPlan),
		Lifecycle: managedfeatures.LifecycleActive,
	})
	require.NoError(t, err)
	return s
}

func (s *sharedServer) all() []pulid.ID {
	return []pulid.ID{
		s.withAI.OrganizationID,
		s.withoutAI.OrganizationID,
		s.suspended.OrganizationID,
		s.noPlan.OrganizationID,
	}
}

func TestKeepOnASharedServerKeepsTheTenantsEntitledToTheNeed(t *testing.T) {
	s := setupSharedServer(t)

	kept, err := s.filter.Keep(s.ctx, entitlement.For(platformcatalog.FeatureAIDesk), s.all())
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{s.withAI.OrganizationID}, kept,
		"only an active plan that lists the add-on; a suspended tenant has nothing")

	for _, need := range []entitlement.Need{
		entitlement.AnyActive(),
		entitlement.For(platformcatalog.FeatureCoreTMS),
	} {
		kept, err = s.filter.Keep(s.ctx, need, s.all())
		require.NoError(t, err)
		assert.Equal(t, []pulid.ID{s.withAI.OrganizationID, s.withoutAI.OrganizationID}, kept,
			"%s: every active plan the control plane sent, and no other", need)
	}

	_, err = s.filter.Keep(s.ctx, entitlement.Need{}, s.all())
	require.ErrorIs(t, err, entitlement.ErrNoNeed)
}

func TestOrganizationsOnASharedServerFiltersAStatementAcrossTenants(t *testing.T) {
	s := setupSharedServer(t)

	st := managedfeaturesdb.States
	read := func(need entitlement.Need) []string {
		t.Helper()
		system := dbscope.WithSystem(s.ctx, "test the entitlement predicate")
		pred, err := s.filter.Organizations(system, st.ScopeKey.Col, need)
		require.NoError(t, err)
		var keys []string
		require.NoError(t, keel.Read(system, s.db, func(ctx context.Context) error {
			var readErr error
			keys, readErr = keel.Into[string](ctx, s.db, st.Pick(st.ScopeKey).Where(pred))
			return readErr
		}))
		slices.Sort(keys)
		return keys
	}

	assert.Equal(t, []string{s.withAI.OrganizationID.String()},
		read(entitlement.For(platformcatalog.FeatureAIDesk)))
	want := []string{s.withAI.OrganizationID.String(), s.withoutAI.OrganizationID.String()}
	slices.Sort(want)
	assert.Equal(t, want, read(entitlement.AnyActive()))
}

func TestForWorkOnASharedServerNeedsTheWorksFeature(t *testing.T) {
	s := setupSharedServer(t)
	st := managedfeaturesdb.States

	_, err := s.filter.ForWork(s.ctx, st.ScopeKey.Col)
	require.ErrorIs(t, err, entitlement.ErrNoWork,
		"a sweep with no feature in its context never reads every organization")

	work := entitlement.WithWork(s.ctx, entitlement.For(platformcatalog.FeatureAIDesk))
	pred, err := s.filter.ForWork(work, st.ScopeKey.Col)
	require.NoError(t, err)
	assert.NotNil(t, pred)
}

func TestADedicatedInstanceAnswersForEveryOrganizationWithNoJoin(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	filter := New(Params{
		DB:     postgres.WrapDB(db),
		Scopes: managedscope.NewResolver(false),
		Gate:   plantest.Without(platformcatalog.FeatureAIDesk),
	})
	organizations := []pulid.ID{pulid.MustNew("org_"), pulid.MustNew("org_")}

	kept, err := filter.Keep(ctx, entitlement.For(platformcatalog.FeatureAIDesk), organizations)
	require.NoError(t, err)
	assert.Empty(t, kept, "the instance's plan lacks the add-on")

	kept, err = filter.Keep(ctx, entitlement.For(platformcatalog.FeatureCoreTMS), organizations)
	require.NoError(t, err)
	assert.Equal(t, organizations, kept)

	st := managedfeaturesdb.States
	system := dbscope.WithSystem(ctx, "test the entitlement predicate")
	_, err = managedfeaturesrepository.New(managedfeaturesrepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	}).Replace(ctx, &managedfeaturesports.ReplaceRequest{
		Scope:    managedscope.Instance(),
		Received: &managedfeatures.Received{Features: []platformcatalog.FeatureKey{}},
		Now:      100,
	})
	require.NoError(t, err)
	matches := func(pred keel.Pred) bool {
		t.Helper()
		var found bool
		require.NoError(t, keel.Read(system, db, func(ctx context.Context) error {
			var readErr error
			found, readErr = st.Where(pred).Exists(ctx, db)
			return readErr
		}))
		return found
	}

	pred, err := filter.ForWork(ctx, st.ScopeKey.Col)
	require.NoError(t, err)
	assert.True(t, matches(pred), "a dedicated sweep with no feature is the instance's own")

	pred, err = filter.Organizations(
		ctx,
		st.ScopeKey.Col,
		entitlement.For(platformcatalog.FeatureAIDesk),
	)
	require.NoError(t, err)
	assert.False(t, matches(pred), "an add-on the instance lacks keeps no organization")
}

func TestEntitledScopesReachTheirIndexesUnderAGenericPlan(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	system := dbscope.WithSystem(ctx, "index plan test")
	st := managedfeaturesdb.States

	cases := []struct {
		name    string
		query   keel.Query[*managedfeatures.State]
		indexes []string
	}{
		{
			name:    "every active plan",
			query:   entitledScopes(entitlement.AnyActive()),
			indexes: []string{"idx_managed_feature_states_entitled"},
		},
		{
			name:  "an add-on",
			query: entitledScopes(entitlement.For(platformcatalog.FeatureAIDesk)),
			indexes: []string{
				"idx_managed_feature_states_entitled",
				"idx_managed_feature_states_active_features",
			},
		},
		{
			name: "a page of organizations",
			query: entitledScopes(entitlement.AnyActive()).
				Where(st.ScopeKey.In("org_a", "org_b")),
			indexes: []string{
				"idx_managed_feature_states_entitled",
				"managed_feature_states_pkey",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var plan *keel.Plan
			err := keel.Read(system, db, func(ctx context.Context) error {
				var err error
				plan, err = tc.query.Explain(ctx, db, keel.ExplainOptions{GenericPlan: true})
				return err
			}, keel.Setting("enable_seqscan", "off"))
			require.NoError(t, err)
			read := plan.Indexes()
			assert.True(t, slices.ContainsFunc(tc.indexes, func(index string) bool {
				return slices.Contains(read, index)
			}), "the generic plan reads %v and scans %v\n%s", read, plan.SeqScans(), plan.JSON)
			assert.Empty(t, plan.SeqScans())
		})
	}
}

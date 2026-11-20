package managedplanservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedplanservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type historyRepo struct {
	managedfeaturesports.StateRepository

	held    *managedfeatures.State
	changes []*managedfeatures.Change
	reads   int
	limit   int
	scope   managedscope.Scope
}

func (r *historyRepo) History(
	_ context.Context,
	scope managedscope.Scope,
	limit int,
) (*managedfeatures.State, []*managedfeatures.Change, error) {
	r.scope = scope
	r.reads++
	r.limit = limit
	return r.held, r.changes, nil
}

type usageReader struct {
	usage *managedusage.Usage
	reads int
	scope managedscope.Scope
}

func (u *usageReader) Usage(
	_ context.Context,
	scope managedscope.Scope,
) (*managedusage.Usage, error) {
	u.scope = scope
	u.reads++
	return u.usage, nil
}

func TestPlanOnASelfHostedInstallReadsNothingAndSaysItIsNotManaged(t *testing.T) {
	t.Parallel()

	repo := &historyRepo{}
	usage := &usageReader{}
	svc := managedplanservice.New(managedplanservice.Params{
		Config: &managedconfig.Config{Enabled: false},
		Repo:   repo,
		Usage:  usage,
		Scopes: managedscope.NewResolver(false),
	})

	plan, err := svc.Plan(t.Context())
	require.NoError(t, err)
	assert.False(t, plan.Managed)
	assert.Empty(t, plan.ManageURL)
	assert.Nil(t, plan.Usage)
	assert.Zero(t, usage.reads)
	assert.Empty(t, plan.View.Included)
	assert.Empty(t, plan.View.Available)
	assert.Empty(t, plan.Changes)
	assert.Zero(t, repo.reads)
}

func TestPlanListsTheAddOnsAndTheRecentChanges(t *testing.T) {
	t.Parallel()

	repo := &historyRepo{
		held: &managedfeatures.State{
			Features: []platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
		},
		changes: []*managedfeatures.Change{{
			Added:     []platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
			Removed:   []platformcatalog.FeatureKey{},
			Starting:  true,
			ChangedAt: 100,
		}},
	}
	svc := managedplanservice.New(managedplanservice.Params{
		Config: &managedconfig.Config{Enabled: true},
		Repo:   repo,
		Usage:  &usageReader{},
		Scopes: managedscope.NewResolver(false),
	})

	plan, err := svc.Plan(t.Context())
	require.NoError(t, err)
	assert.True(t, plan.Managed)
	assert.Equal(t, managedfeatures.PlanHistoryLimit, repo.limit)
	require.Len(t, plan.View.Included, 2)
	assert.Equal(t, "Accounting & Cash", plan.View.Included[0].AddOn.Name)
	assert.Equal(t, "Trenova AI", plan.View.Included[1].AddOn.Name)
	assert.Len(t, plan.View.Available, len(platformcatalog.AddOns())-2)
	require.Len(t, plan.Changes, 1)
	assert.True(t, plan.Changes[0].Starting)
}

func TestPlanBeforeAnyListHasOnlyTheBasePlan(t *testing.T) {
	t.Parallel()

	svc := managedplanservice.New(managedplanservice.Params{
		Config: &managedconfig.Config{Enabled: true},
		Repo:   &historyRepo{},
		Usage:  &usageReader{},
		Scopes: managedscope.NewResolver(false),
	})

	plan, err := svc.Plan(t.Context())
	require.NoError(t, err)
	assert.True(t, plan.Managed)
	assert.Empty(t, plan.View.Included)
	assert.Len(t, plan.View.Available, len(platformcatalog.AddOns()))
	assert.NotEmpty(t, plan.View.Base)
	assert.Empty(t, plan.Changes)
}

func TestPlanCarriesTheManageLinkTheUsageAndScheduledEndings(t *testing.T) {
	t.Parallel()

	usage := &managedusage.Usage{
		PeriodStart: 100,
		PeriodEnd:   200,
		Meters:      []managedusage.MeterUsage{},
	}
	repo := &historyRepo{
		held: &managedfeatures.State{
			Features: []platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
			AddOnEndings: []managedfeatures.AddOnEnding{
				{AddOn: platformcatalog.AddOnAccounting, EndsAt: 900},
			},
			PlanURL: "https://console.trenova.cloud/instances/1/plan",
		},
	}
	svc := managedplanservice.New(managedplanservice.Params{
		Config: &managedconfig.Config{Enabled: true},
		Repo:   repo,
		Usage:  &usageReader{usage: usage},
		Scopes: managedscope.NewResolver(false),
	})

	plan, err := svc.Plan(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "https://console.trenova.cloud/instances/1/plan", plan.ManageURL)
	assert.Same(t, usage, plan.Usage)
	require.Len(t, plan.View.Included, 2)
	assert.Equal(t, int64(900), plan.View.Included[0].EndsAt)
	assert.Zero(t, plan.View.Included[1].EndsAt)
}

func TestPlanOnASharedServerIsTheSignedInOrganizations(t *testing.T) {
	t.Parallel()

	repo := &historyRepo{}
	usage := &usageReader{}
	svc := managedplanservice.New(managedplanservice.Params{
		Config: &managedconfig.Config{Enabled: true, Tenancy: managedconfig.TenancyShared},
		Repo:   repo,
		Usage:  usage,
		Scopes: managedscope.NewResolver(true),
	})
	tenant := dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	_, err := svc.Plan(dbscope.WithTenant(t.Context(), tenant))
	require.NoError(t, err)
	assert.Equal(t, managedscope.Organization(tenant), repo.scope)
	assert.Equal(t, managedscope.Organization(tenant), usage.scope)

	_, err = svc.Plan(t.Context())
	require.ErrorIs(t, err, managedscope.ErrNoTenantScope)
}

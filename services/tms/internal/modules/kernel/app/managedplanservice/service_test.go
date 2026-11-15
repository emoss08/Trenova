package managedplanservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedplanservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type historyRepo struct {
	managedfeaturesports.StateRepository

	held    *managedfeatures.State
	changes []*managedfeatures.Change
	reads   int
	limit   int
}

func (r *historyRepo) History(
	_ context.Context,
	limit int,
) (*managedfeatures.State, []*managedfeatures.Change, error) {
	r.reads++
	r.limit = limit
	return r.held, r.changes, nil
}

func TestPlanOnASelfHostedInstallReadsNothingAndSaysItIsNotManaged(t *testing.T) {
	t.Parallel()

	repo := &historyRepo{}
	svc := managedplanservice.New(managedplanservice.Params{
		Config: &managedconfig.Config{Enabled: false},
		Repo:   repo,
	})

	plan, err := svc.Plan(t.Context())
	require.NoError(t, err)
	assert.False(t, plan.Managed)
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
	})

	plan, err := svc.Plan(t.Context())
	require.NoError(t, err)
	assert.True(t, plan.Managed)
	assert.Empty(t, plan.View.Included)
	assert.Len(t, plan.View.Available, len(platformcatalog.AddOns()))
	assert.NotEmpty(t, plan.View.Base)
	assert.Empty(t, plan.Changes)
}

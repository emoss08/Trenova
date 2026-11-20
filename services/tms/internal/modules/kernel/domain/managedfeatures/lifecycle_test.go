package managedfeatures_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func orgScope() managedscope.Scope {
	return managedscope.Organization(dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func TestASuspendedStateHasNoFeatures(t *testing.T) {
	t.Parallel()

	state := &managedfeatures.State{
		Lifecycle: managedfeatures.LifecycleSuspended,
		Features:  []platformcatalog.FeatureKey{platformcatalog.FeatureAccounting},
	}
	assert.True(t, state.Suspended())
	assert.False(t, state.Has(platformcatalog.FeatureAccounting))

	state.Lifecycle = managedfeatures.LifecycleActive
	assert.True(t, state.Has(platformcatalog.FeatureAccounting))
}

func TestSuspendingAScopeWithNoPlanKeepsItsPlanToCome(t *testing.T) {
	t.Parallel()

	scope := orgScope()
	r := managedfeatures.SetLifecycle(scope, nil, managedfeatures.LifecycleSuspended)
	require.True(t, r.Updated())
	assert.True(t, r.LifecycleChanged())
	assert.Equal(t, scope, r.Scope)
	assert.Equal(t, scope.Key(), r.Current.ScopeKey)
	assert.Equal(t, scope.BusinessUnitID(), r.Current.ScopeBusinessUnitID)
	assert.False(t, r.Current.HasList(), "no list arrived yet")
	assert.Equal(t, managedfeatures.DefaultFinalBackupDays, r.Current.FinalBackupDays)

	next := managedfeatures.Replace(scope, r.Current, &managedfeatures.Received{
		Features: []platformcatalog.FeatureKey{platformcatalog.FeatureAccounting},
	}, 100)
	require.True(t, next.Changed())
	assert.True(t, next.Change.Starting, "the first list is still the starting plan")
	assert.Equal(t, scope.Key(), next.Change.ScopeKey)
	assert.True(t, next.Current.Suspended(), "a list does not lift a suspension")
}

func TestTheSameLifecycleChangesNothing(t *testing.T) {
	t.Parallel()

	scope := orgScope()
	held := &managedfeatures.State{
		ScopeKey:  scope.Key(),
		Lifecycle: managedfeatures.LifecycleActive,
	}
	r := managedfeatures.SetLifecycle(scope, held, managedfeatures.LifecycleActive)
	assert.False(t, r.Updated())

	legacy := &managedfeatures.State{ScopeKey: scope.Key()}
	assert.False(
		t,
		managedfeatures.SetLifecycle(scope, legacy, managedfeatures.LifecycleActive).Updated(),
		"an unset lifecycle reads as active",
	)
}

func TestASuspendedScopeIsNotReminded(t *testing.T) {
	t.Parallel()

	state := &managedfeatures.State{
		Lifecycle:        managedfeatures.LifecycleSuspended,
		PlanEndsAt:       1_000,
		PlanEndAnnounced: 1_000,
	}
	assert.False(t, state.PlanEndReminderDue(900))
}

package dispatchautoassignservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPlan_RefusesWithThePlanRestrictionWithoutDispatchIntelligence(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{}
	svc := New(Params{
		Logger:   zap.NewNop(),
		Planner:  planner,
		Features: plantest.BaseOnly(),
	})

	plan, err := svc.Plan(t.Context(), planRequest())

	require.Nil(t, plan)
	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, string(platformcatalog.FeatureDispatchIntelligence), restriction.Feature)
	assert.Equal(t, "Dispatch Intelligence", restriction.AddOn)
	assert.Empty(t, planner.requests, "nothing is planned before the plan is checked")
}

func TestPlan_RefusesNamingTrenovaAIWhenOnlyDispatchIntelligenceIsOn(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{}
	svc := New(Params{
		Logger:   zap.NewNop(),
		Planner:  planner,
		Features: plantest.Without(platformcatalog.FeatureAIDesk),
	})

	plan, err := svc.Plan(t.Context(), planRequest())

	require.Nil(t, plan)
	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, "Trenova AI", restriction.AddOn)
	assert.Empty(t, planner.requests, "no half-finished plan is made")
}

func TestPlan_PlansWithDispatchIntelligenceAndTrenovaAI(t *testing.T) {
	t.Parallel()

	rig := newPlanningRig(t, planningRigParams{Control: enabledControl(), Planned: solvedPlan()})

	plan, err := rig.svc.Plan(t.Context(), rig.req)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Len(t, rig.planner.requests, 1)
}

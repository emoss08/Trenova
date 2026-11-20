package platformcatalogports_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGate struct {
	disabled map[platformcatalog.FeatureKey]bool
	err      error
	asked    []platformcatalog.FeatureKey
}

func (g *fakeGate) Enabled(_ context.Context, feature platformcatalog.FeatureKey) (bool, error) {
	g.asked = append(g.asked, feature)
	if g.err != nil {
		return false, g.err
	}

	return !g.disabled[feature], nil
}

func (g *fakeGate) EnabledAnywhere(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	return g.Enabled(ctx, feature)
}

func disabling(features ...platformcatalog.FeatureKey) *fakeGate {
	gate := &fakeGate{disabled: map[platformcatalog.FeatureKey]bool{}}
	for _, feature := range features {
		gate.disabled[feature] = true
	}

	return gate
}

func TestRequireFeatureAllowsAnEnabledFeature(t *testing.T) {
	t.Parallel()

	gate := disabling()

	require.NoError(t, platformcatalogports.RequireFeature(
		t.Context(),
		gate,
		platformcatalog.FeatureAIDesk,
	))
	assert.Equal(t, []platformcatalog.FeatureKey{platformcatalog.FeatureAIDesk}, gate.asked)
}

func TestRequireFeatureRefusesADisabledAddOnFeatureNamingTheAddOn(t *testing.T) {
	t.Parallel()

	err := platformcatalogports.RequireFeature(
		t.Context(),
		disabling(platformcatalog.FeatureAIDesk),
		platformcatalog.FeatureAIDesk,
	)

	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, errortypes.ErrPlanRestricted, restriction.GetCode())
	assert.Equal(t, "ai.desk", restriction.Feature)
	assert.Equal(t, "Trenova AI", restriction.AddOn)
	assert.Equal(t, errortypes.PlanRestrictionReasonPlan, restriction.Reason)
	assert.Empty(t, restriction.Plan)
	assert.Equal(
		t,
		"This feature is part of the Trenova AI add-on, which is not on your plan",
		restriction.Error(),
	)
}

func TestRequireFeatureRefusesADisabledBaseFeatureWithoutAnAddOn(t *testing.T) {
	t.Parallel()

	err := platformcatalogports.RequireFeature(
		t.Context(),
		disabling(platformcatalog.FeatureAPIKeys),
		platformcatalog.FeatureAPIKeys,
	)

	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, "platform.api_keys", restriction.Feature)
	assert.Empty(t, restriction.AddOn)
	assert.Equal(t, errortypes.PlanRestrictionReasonPlan, restriction.Reason)
	assert.Equal(
		t,
		"This feature is not available on your organization's plan",
		restriction.Error(),
	)
}

func TestRequireFeatureRejectsAnUnknownFeatureWithoutAskingTheGate(t *testing.T) {
	t.Parallel()

	gate := disabling()

	err := platformcatalogports.RequireFeature(
		t.Context(),
		gate,
		platformcatalog.FeatureKey("teleport"),
	)

	require.ErrorIs(t, err, platformcatalogports.ErrUnknownFeature)
	assert.Equal(t, `unknown platform feature: "teleport"`, err.Error())
	assert.False(t, errortypes.IsPlanRestrictionError(err))
	assert.Empty(t, gate.asked)
}

func TestRequireFeaturePassesTheGatesErrorThrough(t *testing.T) {
	t.Parallel()

	boom := errors.New("purchased list unavailable")

	err := platformcatalogports.RequireFeature(
		t.Context(),
		&fakeGate{err: boom},
		platformcatalog.FeatureCarrierBrokerage,
	)

	require.ErrorIs(t, err, boom)
	assert.False(t, errortypes.IsPlanRestrictionError(err))
}

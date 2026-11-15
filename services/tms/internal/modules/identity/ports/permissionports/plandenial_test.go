package permissionports_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlanDenialNamesTheFeatureAndTheAddOnThatSellsIt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, permissionports.PlanDenial{
		Feature:     platformcatalog.FeatureAIDesk,
		FeatureName: "Desk",
		AddOn:       platformcatalog.AddOnAI,
		AddOnName:   "Trenova AI",
	}, permissionports.NewPlanDenial(platformcatalog.FeatureAIDesk))
}

func TestNewPlanDenialOfABaseFeatureHasNoAddOn(t *testing.T) {
	t.Parallel()

	assert.Equal(t, permissionports.PlanDenial{
		Feature:     platformcatalog.FeatureAPIKeys,
		FeatureName: "API keys",
	}, permissionports.NewPlanDenial(platformcatalog.FeatureAPIKeys))
}

func TestDeniedErrorTurnsAPlanDenialIntoAPlanRestriction(t *testing.T) {
	t.Parallel()

	denial := permissionports.NewPlanDenial(platformcatalog.FeatureAccounting)
	permissionErr := errortypes.NewAuthorizationError("You don't have permission")

	err := permissionports.DeniedError(&permissionports.PermissionCheckResult{
		Reason:     permissionports.ReasonNotOnPlan,
		PlanDenial: &denial,
	}, permissionErr)

	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, errortypes.ErrPlanRestricted, restriction.GetCode())
	assert.Equal(t, "tms.accounting", restriction.Feature)
	assert.Equal(t, "Accounting & Cash", restriction.AddOn)
	assert.Equal(t, errortypes.PlanRestrictionReasonPlan, restriction.Reason)
	assert.Equal(
		t,
		"This feature is part of the Accounting & Cash add-on, which is not on your plan",
		restriction.Error(),
	)
	assert.False(t, errortypes.IsAuthorizationError(err))
}

func TestDeniedErrorKeepsAnOrdinaryDenialAsItWas(t *testing.T) {
	t.Parallel()

	permissionErr := errortypes.NewAuthorizationError("You don't have permission")

	assert.Same(t, permissionErr, permissionports.DeniedError(
		&permissionports.PermissionCheckResult{Reason: "no_permission"},
		permissionErr,
	))
	assert.Same(t, permissionErr, permissionports.DeniedError(nil, permissionErr))
}

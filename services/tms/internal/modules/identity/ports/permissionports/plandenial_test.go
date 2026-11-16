package permissionports_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
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

func TestAPlanDenialReadsInTheReadersLanguage(t *testing.T) {
	t.Parallel()

	denial := permissionports.NewPlanDenial(platformcatalog.FeatureAccounting)

	assert.Equal(t, permissionports.PlanDenial{
		Feature:     platformcatalog.FeatureAccounting,
		FeatureName: "Contabilidad",
		AddOn:       platformcatalog.AddOnAccounting,
		AddOnName:   "Contabilidad y efectivo",
	}, denial.In(i18n.ES))
	assert.Equal(t, denial, denial.In(i18n.EN))
	assert.Equal(t, "Accounting & Cash", denial.AddOnName, "In returns a copy")

	unknown := permissionports.PlanDenial{Feature: "tms.teleport", FeatureName: "Teleport"}
	assert.Equal(t, unknown, unknown.In(i18n.ES), "a name the catalog does not hold stays as written")

	assert.Nil(t, permissionports.DenialsIn(i18n.ES, nil))
	assert.Equal(
		t,
		[]permissionports.PlanDenial{denial.In(i18n.ES)},
		permissionports.DenialsIn(i18n.ES, []permissionports.PlanDenial{denial}),
	)
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

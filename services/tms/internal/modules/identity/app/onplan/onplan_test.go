package onplan_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errPlanUnavailable = errors.New("purchased add-ons unavailable")

type failingGate struct{}

func (failingGate) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, errPlanUnavailable
}

func (r failingGate) EnabledAnywhere(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	return r.Enabled(ctx, feature)
}

type countingGate struct {
	platformcatalogports.FeatureGate

	asked map[platformcatalog.FeatureKey]int
}

func (g *countingGate) Enabled(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	if g.asked == nil {
		g.asked = make(map[platformcatalog.FeatureKey]int, 2)
	}
	g.asked[feature]++
	return g.FeatureGate.Enabled(ctx, feature)
}

func (g *countingGate) EnabledAnywhere(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	return g.Enabled(ctx, feature)
}

func registry(t *testing.T) *permissioncatalog.Registry {
	t.Helper()

	reg := permissioncatalog.NewEmptyRegistry()
	require.NoError(t, reg.Register(&permissionspec.ResourceDefinition{
		Resource: "invoice_probe",
		Feature:  platformcatalog.FeatureBilling,
		FieldFeatures: map[string]platformcatalog.FeatureKey{
			"factoringCompany": platformcatalog.FeatureFactoring,
		},
	}))
	require.NoError(t, reg.Register(&permissionspec.ResourceDefinition{
		Resource: "factoring_probe",
		Feature:  platformcatalog.FeatureFactoring,
	}))

	return reg
}

func TestResource_AnOffFeatureNamesItsAddOn(t *testing.T) {
	t.Parallel()

	plan := onplan.New(registry(t), plantest.Without(platformcatalog.FeatureFactoring))

	denial, off, err := plan.Resource(t.Context(), "factoring_probe")
	require.NoError(t, err)
	assert.True(t, off)
	assert.Equal(t, permissionports.PlanDenial{
		Feature:     platformcatalog.FeatureFactoring,
		FeatureName: "Factoring",
		AddOn:       platformcatalog.AddOnAccounting,
		AddOnName:   "Accounting & Cash",
	}, denial)

	denial, off, err = plan.Resource(t.Context(), "invoice_probe")
	require.NoError(t, err)
	assert.False(t, off)
	assert.Empty(t, denial)
}

func TestResource_AnUnknownResourceIsLeftToTheRoleLogic(t *testing.T) {
	t.Parallel()

	gate := &countingGate{FeatureGate: plantest.Without(platformcatalog.FeatureBilling)}
	plan := onplan.New(registry(t), gate)

	denial, off, err := plan.Resource(t.Context(), "not_a_resource")
	require.NoError(t, err)
	assert.False(t, off)
	assert.Empty(t, denial)
	assert.Empty(t, gate.asked, "nothing is asked of the gate for a resource no feature owns")

	_, off, err = plan.Field(t.Context(), "not_a_resource", "factoringCompany")
	require.NoError(t, err)
	assert.False(t, off)
}

func TestField_IsOffWhenItsResourcesFeatureIsOff(t *testing.T) {
	t.Parallel()

	plan := onplan.New(registry(t), plantest.Without(platformcatalog.FeatureBilling))

	denial, off, err := plan.Field(t.Context(), "invoice_probe", "total")
	require.NoError(t, err)
	assert.True(t, off)
	assert.Equal(t, platformcatalog.FeatureBilling, denial.Feature)

	denial, off, err = plan.Field(t.Context(), "invoice_probe", "factoringCompany")
	require.NoError(t, err)
	assert.True(t, off)
	assert.Equal(t, platformcatalog.FeatureBilling, denial.Feature,
		"the resource's own denial comes first")
}

func TestField_IsOffWhenTheFeatureItDeclaresIsOff(t *testing.T) {
	t.Parallel()

	plan := onplan.New(registry(t), plantest.Without(platformcatalog.FeatureFactoring))

	denial, off, err := plan.Field(t.Context(), "invoice_probe", "factoringCompany")
	require.NoError(t, err)
	assert.True(t, off)
	assert.Equal(t, permissionports.NewPlanDenial(platformcatalog.FeatureFactoring), denial)

	_, off, err = plan.Field(t.Context(), "invoice_probe", "total")
	require.NoError(t, err)
	assert.False(t, off, "a field without a feature of its own follows its resource")
}

func TestField_EveryFeatureOnWithholdsNothing(t *testing.T) {
	t.Parallel()

	plan := onplan.New(registry(t), plantest.AllFeatures())

	for _, field := range []string{"total", "factoringCompany"} {
		_, off, err := plan.Field(t.Context(), "invoice_probe", field)
		require.NoError(t, err)
		assert.Falsef(t, off, "invoice_probe.%s", field)
	}
	unavailable, err := plan.Unavailable(t.Context())
	require.NoError(t, err)
	assert.Empty(t, unavailable)
}

func TestLookup_AGateErrorIsAnError(t *testing.T) {
	t.Parallel()

	plan := onplan.New(registry(t), failingGate{})

	_, off, err := plan.Resource(t.Context(), "invoice_probe")
	require.ErrorIs(t, err, errPlanUnavailable)
	assert.False(t, off)

	_, off, err = plan.Field(t.Context(), "invoice_probe", "factoringCompany")
	require.ErrorIs(t, err, errPlanUnavailable)
	assert.False(t, off)

	_, err = plan.Unavailable(t.Context())
	require.ErrorIs(t, err, errPlanUnavailable)
}

func TestUnavailable_ListsEveryOffFeatureInCatalogOrder(t *testing.T) {
	t.Parallel()

	plan := onplan.New(registry(t), plantest.Without(
		platformcatalog.FeatureFactoring,
		platformcatalog.FeatureAccounting,
	))

	unavailable, err := plan.Unavailable(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []permissionports.PlanDenial{
		permissionports.NewPlanDenial(platformcatalog.FeatureAccounting),
		permissionports.NewPlanDenial(platformcatalog.FeatureFactoring),
	}, unavailable)
}

func TestMemo_AsksTheGateOncePerFeature(t *testing.T) {
	t.Parallel()

	gate := &countingGate{FeatureGate: plantest.Without(platformcatalog.FeatureFactoring)}
	memo := onplan.New(registry(t), gate).Memo()

	for range 3 {
		_, _, err := memo.Resource(t.Context(), "factoring_probe")
		require.NoError(t, err)
		_, _, err = memo.Field(t.Context(), "invoice_probe", "factoringCompany")
		require.NoError(t, err)
	}

	assert.Equal(t, map[platformcatalog.FeatureKey]int{
		platformcatalog.FeatureFactoring: 1,
		platformcatalog.FeatureBilling:   1,
	}, gate.asked)
}

package refview_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func everyFeature(registry *permissioncatalog.Registry) *onplan.Lookup {
	if registry == nil {
		registry = permissioncatalog.NewRegistry()
	}
	return onplan.New(registry, plantest.AllFeatures())
}

func allOn() *refview.Views {
	registry := permissioncatalog.NewRegistry()
	return refview.New(registry, everyFeature(registry))
}

const (
	invoiceKind   permission.Resource = "invoice_probe"
	factoringKind permission.Resource = "factoring_probe"
	shipmentKind  permission.Resource = "shipment_probe"
	detentionKind permission.Resource = "detention_probe"
	stopKind      permission.Resource = "stop_probe"
)

// planRegistry is a base invoice that carries an add-on's factoring company
// as its own field, the factoring company as a kind of that add-on, and a
// base shipment with a part of its own family in an add-on.
func planRegistry(t *testing.T) *permissioncatalog.Registry {
	t.Helper()

	registry := permissioncatalog.NewEmptyRegistry()
	for _, def := range []*permissionspec.ResourceDefinition{
		{
			Resource:      invoiceKind.String(),
			Feature:       platformcatalog.FeatureBilling,
			ReferenceView: []string{"number"},
			FieldFeatures: map[string]platformcatalog.FeatureKey{
				"factoringCompany": platformcatalog.FeatureFactoring,
			},
		},
		{
			Resource:      factoringKind.String(),
			Feature:       platformcatalog.FeatureFactoring,
			ReferenceView: []string{"name"},
		},
		{
			Resource:      shipmentKind.String(),
			Feature:       platformcatalog.FeatureCoreTMS,
			ReferenceView: []string{"proNumber"},
		},
		{
			Resource:       detentionKind.String(),
			Feature:        platformcatalog.FeatureDetention,
			ParentResource: shipmentKind.String(),
		},
		{
			Resource:       stopKind.String(),
			Feature:        platformcatalog.FeatureCoreTMS,
			ParentResource: shipmentKind.String(),
		},
	} {
		require.NoError(t, registry.Register(def))
	}

	return registry
}

func planViews(t *testing.T, gate platformcatalogports.FeatureGate) *refview.Views {
	t.Helper()

	registry := planRegistry(t)
	return refview.New(registry, onplan.New(registry, gate))
}

type failingGate struct{}

func (failingGate) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, errors.New("the plan is unavailable")
}

func readsEverything() access {
	return access{reads: map[permission.Resource]bool{
		invoiceKind:   true,
		factoringKind: true,
		shipmentKind:  true,
		detentionKind: true,
		stopKind:      true,
	}}
}

var reaches = []refview.Reach{refview.Root, refview.Nested, refview.SelfScoped}

func TestDecide_AKindOffThePlanWithholdsEveryFieldAtEveryReach(t *testing.T) {
	t.Parallel()

	views := planViews(t, plantest.Without(platformcatalog.FeatureFactoring))
	for _, reach := range reaches {
		for _, field := range []string{"name", "remitTo", "id"} {
			assert.Equalf(t, refview.WithheldPlan,
				views.Decide(t.Context(), readsEverything(), factoringKind, field, reach),
				"%s reached as %d, for a reader of every kind", field, reach)
		}
	}
}

func TestDecide_AFieldOfAnotherFeatureIsWithheldOnARootRecordOfAReadableKind(t *testing.T) {
	t.Parallel()

	views := planViews(t, plantest.Without(platformcatalog.FeatureFactoring))

	assert.Equal(t, refview.WithheldPlan, views.Decide(
		t.Context(), readsEverything(), invoiceKind, "factoringCompany", refview.Root))
	assert.Equal(t, refview.Visible, views.Decide(
		t.Context(), readsEverything(), invoiceKind, "total", refview.Root),
		"the invoice's own fields stay")
	assert.Equal(t, refview.Visible, views.Decide(
		t.Context(), access{}, invoiceKind, "number", refview.Nested),
		"its reference view stays for a reader who reaches it through another record")
}

func TestDecide_ThePlanIsAskedBeforeReadAndCeiling(t *testing.T) {
	t.Parallel()

	views := planViews(t, plantest.Without(platformcatalog.FeatureFactoring))
	nobody := access{ceiling: permission.SensitivityPublic, applies: true}

	assert.Equal(t, refview.WithheldPlan,
		views.Decide(t.Context(), nobody, factoringKind, "remitTo", refview.Nested),
		"neither the missing read nor the ceiling is the reason")
}

func TestDecide_AGateErrorWithholdsTheFieldWithoutNamingThePlan(t *testing.T) {
	t.Parallel()

	views := planViews(t, failingGate{})

	assert.Equal(t, refview.WithheldKind, views.Decide(
		t.Context(), readsEverything(), invoiceKind, "total", refview.Root))
	assert.Equal(t, refview.WithheldKind, views.DecidePart(
		t.Context(), readsEverything(), stopKind, shipmentKind, "city", refview.Root))
	_, off, err := views.PlanDenial(t.Context(), invoiceKind, "total")
	require.Error(t, err, "a plan that could not be read is an error, never a plan withholding")
	assert.False(t, off)
}

func TestDecide_EveryFeatureOnAnswersAsWithoutThePlan(t *testing.T) {
	t.Parallel()

	views := planViews(t, plantest.AllFeatures())
	reader := readsEverything()
	none := access{}
	cases := []struct {
		access refview.Access
		kind   permission.Resource
		field  string
		reach  refview.Reach
		want   refview.Decision
	}{
		{none, factoringKind, "name", refview.Nested, refview.Visible},
		{none, factoringKind, "remitTo", refview.Nested, refview.WithheldKind},
		{none, factoringKind, "remitTo", refview.Root, refview.Visible},
		{reader, factoringKind, "remitTo", refview.Nested, refview.Visible},
		{none, invoiceKind, "factoringCompany", refview.Root, refview.Visible},
		{none, invoiceKind, "factoringCompany", refview.Nested, refview.WithheldKind},
		{none, invoiceKind, "id", refview.Nested, refview.Visible},
	}
	for _, tc := range cases {
		assert.Equalf(t, tc.want,
			views.Decide(t.Context(), tc.access, tc.kind, tc.field, tc.reach),
			"%s.%s reached as %d", tc.kind, tc.field, tc.reach)
	}
	_, off, err := views.PlanDenial(t.Context(), factoringKind, "name")
	require.NoError(t, err)
	assert.False(t, off)
}

func TestDecidePart_APartOfAnAddOnKindUnderABaseCarrierIsWithheld(t *testing.T) {
	t.Parallel()

	views := planViews(t, plantest.Without(platformcatalog.FeatureDetention))

	for _, reach := range reaches {
		assert.Equal(t, refview.WithheldPlan, views.DecidePart(
			t.Context(), readsEverything(), detentionKind, shipmentKind, "billableMinutes", reach))
	}
	assert.Equal(t, refview.Visible, views.DecidePart(
		t.Context(), readsEverything(), stopKind, shipmentKind, "city", refview.Nested),
		"a base part of the same carrier stays")

	denial, off, err := views.PartDenial(
		t.Context(), detentionKind, shipmentKind, "billableMinutes")
	require.NoError(t, err)
	require.True(t, off)
	assert.Equal(t, permissionports.NewPlanDenial(platformcatalog.FeatureDetention), denial)
}

func TestDecidePart_APartOfACarrierOffThePlanIsWithheld(t *testing.T) {
	t.Parallel()

	views := planViews(t, plantest.Without(platformcatalog.FeatureCoreTMS))

	assert.Equal(t, refview.WithheldPlan, views.DecidePart(
		t.Context(), readsEverything(), detentionKind, shipmentKind, "billableMinutes",
		refview.Nested))
	denial, off, err := views.PartDenial(
		t.Context(), detentionKind, shipmentKind, "billableMinutes")
	require.NoError(t, err)
	require.True(t, off)
	assert.Equal(t, platformcatalog.FeatureCoreTMS, denial.Feature)
}

func TestDecidePart_EveryFeatureOnFollowsTheCarriersRead(t *testing.T) {
	t.Parallel()

	views := planViews(t, plantest.AllFeatures())
	none := access{}

	assert.Equal(t, refview.WithheldKind, views.DecidePart(
		t.Context(), none, detentionKind, shipmentKind, "billableMinutes", refview.Nested))
	assert.Equal(t, refview.Visible, views.DecidePart(
		t.Context(), access{reads: map[permission.Resource]bool{shipmentKind: true}},
		detentionKind, shipmentKind, "billableMinutes", refview.Nested))
	assert.Equal(t, refview.Visible, views.DecidePart(
		t.Context(), none, detentionKind, shipmentKind, "billableMinutes", refview.Root))
}

func TestFilter_TellsPlanWithheldFieldsFromAccessWithheldOnes(t *testing.T) {
	t.Parallel()

	views := planViews(t, plantest.Without(platformcatalog.FeatureFactoring))
	filtered := views.Filter(t.Context(), access{}, invoiceKind,
		[]string{"number", "factoringCompany", "total"}, refview.Nested)

	assert.Equal(t, []string{"number"}, filtered.Visible)
	assert.Equal(t, []string{"total"}, filtered.Withheld)
	assert.Equal(t, []refview.PlanField{{
		Field:  "factoringCompany",
		Denial: permissionports.NewPlanDenial(platformcatalog.FeatureFactoring),
	}}, filtered.WithheldByPlan)
}

func TestRequest_AsksThePlanOncePerFeature(t *testing.T) {
	t.Parallel()

	gate := &countingGate{Gate: plantest.Without(platformcatalog.FeatureFactoring)}
	registry := planRegistry(t)
	views := refview.New(registry, onplan.New(registry, gate)).Request()

	for range 3 {
		views.Decide(t.Context(), access{}, factoringKind, "name", refview.Nested)
		views.Decide(t.Context(), access{}, factoringKind, "remitTo", refview.Nested)
	}

	assert.Equal(t, 1, gate.asked[platformcatalog.FeatureFactoring])
}

type countingGate struct {
	*plantest.Gate

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
	return g.Gate.Enabled(ctx, feature)
}

func TestPort_MapsAPlanWithholdingAndNamesItsFeature(t *testing.T) {
	t.Parallel()

	port := refview.NewPort(planViews(t, plantest.Without(platformcatalog.FeatureFactoring)), nil)
	linked := port.Person(
		permissionports.UserActor(pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		}),
		permissionports.PersonGrants{
			Reads: func(context.Context, permission.Resource) (bool, error) { return true, nil },
		},
	)

	assert.Equal(t, permissionports.LinkedFieldWithheldPlan,
		linked.Decide(t.Context(), factoringKind, "name"))
	assert.Equal(t, permissionports.LinkedFieldWithheldPlan,
		linked.Decide(t.Context(), invoiceKind, "factoringCompany"))
	assert.Equal(t, permissionports.LinkedFieldVisible,
		linked.Decide(t.Context(), invoiceKind, "total"))

	denial, off, err := linked.PlanDenial(t.Context(), factoringKind, "name")
	require.NoError(t, err)
	require.True(t, off)
	assert.Equal(t, permissionports.NewPlanDenial(platformcatalog.FeatureFactoring), denial)
	_, off, err = linked.PlanDenial(t.Context(), invoiceKind, "")
	require.NoError(t, err)
	assert.False(t, off, "an empty field asks about the kind alone")
	_, off, err = linked.PlanDenial(t.Context(), invoiceKind, "factoringCompany")
	require.NoError(t, err)
	assert.True(t, off)
}

func TestRoleAccess_ReadsNoKindThePlanLeavesOut(t *testing.T) {
	t.Parallel()

	registry := planRegistry(t)
	grants := permissioncatalog.GrantsOf(role(
		permission.SensitivityInternal,
		grant(factoringKind, permission.OpRead),
		grant(invoiceKind, permission.OpRead),
	))
	off := refview.NewRoleAccess(registry,
		onplan.New(registry, plantest.Without(platformcatalog.FeatureFactoring)), grants)
	on := refview.NewRoleAccess(registry, onplan.New(registry, plantest.AllFeatures()), grants)
	broken := refview.NewRoleAccess(registry, onplan.New(registry, failingGate{}), grants)

	assert.False(t, off.Reads(t.Context(), factoringKind))
	assert.False(t, off.Allows(t.Context(), factoringKind.String(), permission.OpRead))
	assert.True(t, off.Reads(t.Context(), invoiceKind))
	assert.True(t, on.Reads(t.Context(), factoringKind))
	assert.False(t, broken.Reads(t.Context(), invoiceKind),
		"a plan that cannot be read allows nothing")
}

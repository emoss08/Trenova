package gqlext

import (
	"context"
	"testing"

	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	planInvoice   permission.Resource = "invoice"
	planFactoring permission.Resource = "factoring_company"
	planCustomer  permission.Resource = "customer"
)

// planGuard is the guard over a small catalog: a base invoice whose
// factoringStatus belongs to factoring, a factoring company of the factoring
// add-on, and a base customer, read by a person whose role reads every kind
// but the denied ones.
func planGuard(
	t *testing.T,
	gate platformcatalogports.FeatureGate,
	denied ...permission.Resource,
) (*FieldGuardExtension, context.Context) {
	t.Helper()

	registry := permissioncatalog.NewEmptyRegistry()
	for _, def := range []*permissionspec.ResourceDefinition{
		{
			Resource:      planInvoice.String(),
			Feature:       platformcatalog.FeatureBilling,
			ReferenceView: []string{"number"},
			FieldFeatures: map[string]platformcatalog.FeatureKey{
				"factoringStatus": platformcatalog.FeatureFactoring,
			},
		},
		{
			Resource:      planFactoring.String(),
			Feature:       platformcatalog.FeatureFactoring,
			ReferenceView: []string{"name"},
		},
		{
			Resource:      planCustomer.String(),
			Feature:       platformcatalog.FeatureCoreTMS,
			ReferenceView: []string{"name"},
		},
	} {
		require.NoError(t, registry.Register(def))
	}

	names := make([]string, 0, len(denied))
	for _, kind := range denied {
		names = append(names, kind.String())
	}
	guard := newFieldGuard(t, &mocks.DenyResourcesPermissionEngine{Denied: names})
	guard.registry = registry
	guard.views = refview.New(registry, onplan.New(registry, gate))
	guard.resources = map[string]permission.Resource{
		"Invoice":          planInvoice,
		"FactoringCompany": planFactoring,
		"Customer":         planCustomer,
	}
	guard.returns = map[string]permission.Resource{
		"Invoice":                    planInvoice,
		"FactoringCompany":           planFactoring,
		"FactoringCompanyConnection": planFactoring,
		"Customer":                   planCustomer,
	}
	guard.wrappers = map[string]struct{}{}

	return guard, operationContext(t, guard)
}

func withheldByPlan(ctx context.Context) []gqlctx.WithheldByPlan {
	holder, _ := ctx.Value(fieldVisibilityKey{}).(*fieldVisibility)
	if holder == nil {
		return nil
	}
	return holder.withheldByPlan()
}

var withoutFactoring = plantest.Without(platformcatalog.FeatureFactoring)

var invoiceFactor = []fieldStep{
	{object: "Query", field: "invoice", returns: "Invoice"},
	{object: "Invoice", field: "factoringCompany", returns: "FactoringCompany"},
}

func refuseThrough(
	t *testing.T,
	guard *FieldGuardExtension,
	ctx context.Context,
	steps ...fieldStep,
) (bool, error) {
	t.Helper()

	resolved := false
	_, err := guard.InterceptField(nestedFieldContext(ctx, steps...),
		func(context.Context) (any, error) {
			resolved = true
			return "stored value", nil
		})
	return resolved, err
}

func requirePlanRestricted(t *testing.T, err error, feature platformcatalog.FeatureKey) {
	t.Helper()

	var restricted *errortypes.PlanRestrictionError
	require.ErrorAs(t, err, &restricted)
	assert.Equal(t, errortypes.ErrPlanRestricted, restricted.Code)
	assert.Equal(t, string(feature), restricted.Feature)
	assert.Equal(t, "Accounting & Cash", restricted.AddOn)
}

func TestPlanRead_ANestedRecordOffThePlanIsNullAndItsResolverNeverRuns(t *testing.T) {
	t.Parallel()

	guard, ctx := planGuard(t, withoutFactoring)

	result, resolved := resolveNested(t, guard, ctx, invoiceFactor...)

	assert.Nil(t, result)
	assert.False(t, resolved, "the factoring company is never loaded")
	assert.Equal(t, []gqlctx.WithheldByPlan{{
		Path: "invoice.factoringCompany", Feature: platformcatalog.FeatureFactoring,
	}}, withheldByPlan(ctx))
	assert.Empty(t, withheldPaths(ctx), "a plan withholding is not an access withholding")
}

func TestPlanRead_AnAdministratorWhoReadsEveryKindIsStillWithheld(t *testing.T) {
	t.Parallel()

	guard, ctx := planGuard(t, withoutFactoring)
	guard.engine = &mocks.AllowAllPermissionEngine{}

	result, resolved := resolveNested(t, guard, ctx, invoiceFactor...)
	assert.Nil(t, result)
	assert.False(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "search", returns: "SearchHit"},
		fieldStep{object: "FactoringCompany", field: "name"},
	)
	assert.Nil(t, result, "a record reached through an interface is withheld field by field")
	assert.False(t, resolved)
	assert.Equal(t, []gqlctx.WithheldByPlan{
		{Path: "invoice.factoringCompany", Feature: platformcatalog.FeatureFactoring},
		{Path: "search.name", Feature: platformcatalog.FeatureFactoring},
	}, withheldByPlan(ctx))
}

func TestPlanRead_ARootFieldReturningAKindOffThePlanIsRefused(t *testing.T) {
	t.Parallel()

	guard, ctx := planGuard(t, withoutFactoring)

	resolved, err := refuseThrough(t, guard, ctx,
		fieldStep{object: "Query", field: "factoringCompany", returns: "FactoringCompany"})
	requirePlanRestricted(t, err, platformcatalog.FeatureFactoring)
	assert.False(t, resolved)

	resolved, err = refuseThrough(t, guard, ctx, fieldStep{
		object: "Query", field: "factoringCompanies", returns: "FactoringCompanyConnection",
		nonNull: true,
	})
	requirePlanRestricted(t, err, platformcatalog.FeatureFactoring)
	assert.False(t, resolved, "a connection of the kind is refused as the kind is")
	assert.Empty(t, withheldByPlan(ctx), "a refusal is an error, not a withholding")
}

func TestPlanRead_ANonNullNestedRecordOffThePlanIsRefused(t *testing.T) {
	t.Parallel()

	guard, ctx := planGuard(t, withoutFactoring)

	resolved, err := refuseThrough(t, guard, ctx,
		fieldStep{object: "Query", field: "invoice", returns: "Invoice"},
		fieldStep{
			object: "Invoice", field: "factoringCompany", returns: "FactoringCompany",
			nonNull: true,
		},
	)
	requirePlanRestricted(t, err, platformcatalog.FeatureFactoring)
	assert.False(t, resolved)
}

func TestPlanRead_AFieldOfAFeatureOffThePlanIsNullOnARootRecord(t *testing.T) {
	t.Parallel()

	guard, ctx := planGuard(t, withoutFactoring)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "invoice", returns: "Invoice"},
		fieldStep{object: "Invoice", field: "factoringStatus"},
	)
	assert.Nil(t, result)
	assert.False(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "invoice", returns: "Invoice"},
		fieldStep{object: "Invoice", field: "number"},
	)
	assert.Equal(t, "stored value", result, "the invoice's own fields stay")
	assert.True(t, resolved)

	assert.Equal(t, []gqlctx.WithheldByPlan{{
		Path: "invoice.factoringStatus", Feature: platformcatalog.FeatureFactoring,
	}}, withheldByPlan(ctx))
}

func TestPlanRead_ANonNullFieldOfAFeatureOffThePlanIsRefusedForAPerson(t *testing.T) {
	t.Parallel()

	guard, ctx := planGuard(t, withoutFactoring)

	resolved, err := refuseThrough(t, guard, ctx,
		fieldStep{object: "Query", field: "invoice", returns: "Invoice"},
		fieldStep{object: "Invoice", field: "factoringStatus", nonNull: true},
	)
	requirePlanRestricted(t, err, platformcatalog.FeatureFactoring)
	assert.NotContains(t, err.Error(), "@requiresRead")
	assert.False(t, resolved)
}

func TestPlanRead_TheResponseNamesPlanWithholdingsApartFromAccessOnes(t *testing.T) {
	t.Parallel()

	guard, ctx := planGuard(t, withoutFactoring, planCustomer)

	resolveNested(t, guard, ctx, invoiceFactor...)
	resolveNested(t, guard, ctx, invoiceFactor...)
	resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "invoice", returns: "Invoice"},
		fieldStep{object: "Invoice", field: "customer", returns: "Customer"},
		fieldStep{object: "Customer", field: "creditLimit"},
	)

	resp := guard.InterceptResponse(ctx, func(context.Context) *gqlgen.Response {
		return &gqlgen.Response{}
	})
	assert.Equal(t, []gqlctx.WithheldByPlan{{
		Path: "invoice.factoringCompany", Feature: platformcatalog.FeatureFactoring,
	}}, resp.Extensions[withheldByPlanExtension], "sorted, without duplicates")
	assert.Equal(t, []string{"invoice.customer.creditLimit"}, resp.Extensions[withheldExtension])
}

func TestPlanRead_AnAgentReadRecordsThePlanWithholdingWithItsFeature(t *testing.T) {
	t.Parallel()

	guard, _ := planGuard(t, withoutFactoring)
	ceiling := gqlctx.NewFieldCeiling(fixedCeilings(permission.SensitivityRestricted))
	ctx := gqlctx.WithFieldCeiling(operationContext(t, guard), ceiling)

	result, resolved := resolveNested(t, guard, ctx, invoiceFactor...)
	assert.Nil(t, result)
	assert.False(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "invoice", returns: "Invoice"},
		fieldStep{object: "Invoice", field: "factoringStatus", nonNull: true},
	)
	assert.Equal(t, "stored value", result, "nulling a non-null field would null its parent")
	assert.True(t, resolved)

	assert.Equal(t, []gqlctx.WithheldByPlan{
		{Path: "invoice.factoringCompany", Feature: platformcatalog.FeatureFactoring},
		{Path: "invoice.factoringStatus", Feature: platformcatalog.FeatureFactoring},
	}, ceiling.WithheldByPlan())
	assert.Empty(t, ceiling.Withheld(), "the plan's withholdings are not the access's")
	require.Len(t, ceiling.Stripped(), 1)
	assert.Equal(t, "invoice.factoringStatus", ceiling.Stripped()[0].String())
}

func TestPlanRead_EveryFeatureOnAnswersAsBefore(t *testing.T) {
	t.Parallel()

	guard, ctx := planGuard(t, plantest.AllFeatures(), planFactoring)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "invoice", returns: "Invoice"},
		fieldStep{object: "Invoice", field: "factoringCompany", returns: "FactoringCompany"},
		fieldStep{object: "FactoringCompany", field: "name"},
	)
	assert.Equal(t, "stored value", result, "the view of a linked factoring company shows")
	assert.True(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "invoice", returns: "Invoice"},
		fieldStep{object: "Invoice", field: "factoringCompany", returns: "FactoringCompany"},
		fieldStep{object: "FactoringCompany", field: "remitTo"},
	)
	assert.Nil(t, result, "the rest needs factoring company read")
	assert.False(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "invoice", returns: "Invoice"},
		fieldStep{object: "Invoice", field: "factoringStatus"},
	)
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)

	assert.Empty(t, withheldByPlan(ctx))
	assert.Equal(t, []string{"invoice.factoringCompany.remitTo"}, withheldPaths(ctx))
}

package gqlext

import (
	"context"
	"strings"
	"testing"

	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

// schemaPlanGuard is the guard over the real schema and registry, read by a
// person whose role reads every kind, on the plan the gate describes.
func schemaPlanGuard(
	t *testing.T,
	gate platformcatalogports.FeatureGate,
) (*FieldGuardExtension, *ast.Schema, context.Context) {
	t.Helper()

	schema := loadSchema(t)
	guard := newFieldGuard(t, &mocks.AllowAllPermissionEngine{})
	resources, err := KindTypes(schema, guard.registry)
	require.NoError(t, err)
	guard.resources = resources
	guard.returns = RecordTypes(schema, resources)
	guard.wrappers = WrapperTypes(schema)
	guard.views = refview.New(guard.registry, onplan.New(guard.registry, gate))

	return guard, schema, operationContext(t, guard)
}

// schemaFields is the field contexts of a path through the real schema, from
// a root field down, each field carrying its schema definition.
func schemaFields(
	t *testing.T,
	schema *ast.Schema,
	ctx context.Context,
	path ...string,
) context.Context {
	t.Helper()

	for _, step := range path {
		object, field, ok := strings.Cut(step, ".")
		require.True(t, ok, "%s is not Type.field", step)
		def := schema.Types[object]
		require.NotNil(t, def, "the schema has no %s", object)
		fieldDef := def.Fields.ForName(field)
		require.NotNil(t, fieldDef, "the schema has no %s", step)
		ctx = gqlgen.WithFieldContext(ctx, &gqlgen.FieldContext{
			Object: object,
			Field: gqlgen.CollectedField{Field: &ast.Field{
				Name:       field,
				Alias:      field,
				Definition: fieldDef,
			}},
		})
	}

	return ctx
}

// addOnRecordOnBaseRecord is a field of a record of a base feature, or of a
// page about one, that carries a record of an add-on's kind, with the path a
// screen reaches it by and the feature its kind belongs to.
type addOnRecordOnBaseRecord struct {
	path    []string
	feature platformcatalog.FeatureKey
	answer  string
}

var addOnRecordsOnBaseRecords = []addOnRecordOnBaseRecord{
	{
		path:    []string{"Query.invoice", "Invoice.factoringCompany"},
		feature: platformcatalog.FeatureFactoring,
		answer:  "invoice.factoringCompany",
	},
	{
		path:    []string{"Query.invoice", "Invoice.factoringState"},
		feature: platformcatalog.FeatureFactoring,
		answer:  "invoice.factoringState",
	},
	{
		path:    []string{"Query.invoice", "Invoice.paymentApplications"},
		feature: platformcatalog.FeatureAccounting,
		answer:  "invoice.paymentApplications",
	},
	{
		path:    []string{"Query.invoice", "Invoice.creditApplications"},
		feature: platformcatalog.FeatureAccounting,
		answer:  "invoice.creditApplications",
	},
	{
		path:    []string{"Query.invoice", "Invoice.lateChargeAssessments"},
		feature: platformcatalog.FeatureAccounting,
		answer:  "invoice.lateChargeAssessments",
	},
	{
		path:    []string{"Query.invoice", "Invoice.ediSendPlan"},
		feature: platformcatalog.FeatureEDIIntegration,
		answer:  "invoice.ediSendPlan",
	},
	{
		path: []string{
			"Query.dispatchBoard", "DispatchBoard.moves", "DispatchBoardMove.liveTender",
		},
		feature: platformcatalog.FeatureCarrierBrokerage,
		answer:  "dispatchBoard.moves.liveTender",
	},
	{
		path:    []string{"Query.shipmentAnalytics", "ShipmentAnalytics.detentionWatchlist"},
		feature: platformcatalog.FeatureDetention,
		answer:  "shipmentAnalytics.detentionWatchlist",
	},
	{
		path:    []string{"Query.worker", "Worker.pto"},
		feature: platformcatalog.FeatureWorkforceTimeOff,
		answer:  "worker.pto",
	},
	{
		path:    []string{"Query.customer", "Customer.brokerIntelligence"},
		feature: platformcatalog.FeatureCarrierBrokerage,
		answer:  "customer.brokerIntelligence",
	},
	{
		path:    []string{"Query.carrier", "Carrier.intelligence"},
		feature: platformcatalog.FeatureCarrierBrokerage,
		answer:  "carrier.intelligence",
	},
	{
		path:    []string{"Query.shipment", "Shipment.customer", "ShipmentCustomer.ediPartner"},
		feature: platformcatalog.FeatureEDIIntegration,
		answer:  "shipment.customer.ediPartner",
	},
	{
		path:    []string{"Query.role", "Role.agents"},
		feature: platformcatalog.FeatureAgentAutomation,
		answer:  "role.agents",
	},
}

func interceptSchemaField(
	t *testing.T,
	guard *FieldGuardExtension,
	schema *ast.Schema,
	ctx context.Context,
	path ...string,
) (any, bool) {
	t.Helper()

	resolved := false
	result, err := guard.InterceptField(schemaFields(t, schema, ctx, path...),
		func(context.Context) (any, error) {
			resolved = true
			return "stored value", nil
		})
	require.NoError(t, err, "%s is refused instead of answered null", strings.Join(path, " > "))

	return result, resolved
}

// On a plan of the base features alone, each record of an add-on that a base
// record carries is answered null without its resolver running, and the
// response names it with its feature; the base record itself still answers.
func TestPlanSchema_ABaseOnlyPlanWithholdsEveryAddOnRecordABaseRecordCarries(t *testing.T) {
	t.Parallel()

	for _, tc := range addOnRecordsOnBaseRecords {
		t.Run(tc.answer, func(t *testing.T) {
			t.Parallel()

			guard, schema, ctx := schemaPlanGuard(t, plantest.BaseOnly())

			result, resolved := interceptSchemaField(t, guard, schema, ctx, tc.path...)

			assert.Nil(t, result)
			assert.False(t, resolved, "the add-on's record is never loaded")
			assert.Equal(t, []gqlctx.WithheldByPlan{{Path: tc.answer, Feature: tc.feature}},
				withheldByPlan(ctx))
			assert.Empty(t, withheldPaths(ctx), "a plan withholding is not an access withholding")
		})
	}
}

func TestPlanSchema_ABaseOnlyPlanStillAnswersTheBaseRecord(t *testing.T) {
	t.Parallel()

	guard, schema, ctx := schemaPlanGuard(t, plantest.BaseOnly())

	result, resolved := interceptSchemaField(t, guard, schema, ctx,
		"Query.invoice", "Invoice.number")

	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
	assert.Empty(t, withheldByPlan(ctx))
}

// With every feature on, the same fields are answered as they were before
// the plan was applied.
func TestPlanSchema_EveryFeatureOnAnswersEveryAddOnRecordAsBefore(t *testing.T) {
	t.Parallel()

	for _, tc := range addOnRecordsOnBaseRecords {
		t.Run(tc.answer, func(t *testing.T) {
			t.Parallel()

			guard, schema, ctx := schemaPlanGuard(t, plantest.AllFeatures())

			result, resolved := interceptSchemaField(t, guard, schema, ctx, tc.path...)

			assert.Equal(t, "stored value", result)
			assert.True(t, resolved)
			assert.Empty(t, withheldByPlan(ctx))
			assert.Empty(t, withheldPaths(ctx))
		})
	}
}

// The fields above are nullable, as every field carrying another add-on's
// record must be: a non-null one would refuse the whole operation instead.
func TestPlanSchema_EveryAddOnRecordABaseRecordCarriesIsNullable(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	for _, tc := range addOnRecordsOnBaseRecords {
		last := tc.path[len(tc.path)-1]
		object, field, _ := strings.Cut(last, ".")
		def := schema.Types[object].Fields.ForName(field)
		require.NotNil(t, def, last)
		assert.False(t, def.Type.NonNull, "%s must be nullable", last)
	}
}

// A field that declares @authorizes returns a record of its kind only to a
// caller who reads it, so the guard shows the record whole: a customer's
// broker vetting, whose resolver checks carrier intelligence read. An add-on's
// record a base record carries with no such field is linked: an invoice's
// payment application shows its amounts only with customer payment read.
func TestPlanSchema_AnAddOnRecordIsLinkedUnlessItsFieldAuthorizesIt(t *testing.T) {
	t.Parallel()

	guard, schema, ctx := schemaPlanGuard(t, plantest.AllFeatures())
	guard.engine = &mocks.DenyResourcesPermissionEngine{Denied: []string{
		"carrier_intelligence", "customer_payment",
	}}

	result, resolved := interceptSchemaField(t, guard, schema, ctx,
		"Query.customer", "Customer.brokerIntelligence", "CarrierIntelSnapshot.riskLevel")
	assert.Equal(t, "stored value", result, "the resolver answered it, so it is shown")
	assert.True(t, resolved)

	result, resolved = interceptSchemaField(t, guard, schema, ctx,
		"Query.invoice", "Invoice.paymentApplications",
		"CustomerPaymentApplication.appliedAmountMinor")
	assert.Nil(t, result)
	assert.False(t, resolved)

	assert.Equal(t, []string{"invoice.paymentApplications.appliedAmountMinor"},
		withheldPaths(ctx))
	assert.Empty(t, withheldByPlan(ctx))
}

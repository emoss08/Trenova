package gqlext

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
)

// addOnValuesOnBaseRecords are plain values of an add-on stored on a base
// record (FieldFeatures), each with the path a screen reads it by.
var addOnValuesOnBaseRecords = []addOnRecordOnBaseRecord{
	{
		path:    []string{"Query.shipment", "Shipment.tenderStatus"},
		feature: platformcatalog.FeatureCarrierBrokerage,
		answer:  "shipment.tenderStatus",
	},
	{
		path:    []string{"Query.shipment", "Shipment.eta"},
		feature: platformcatalog.FeatureCustomerTracking,
		answer:  "shipment.eta",
	},
	{
		path: []string{
			"Query.shipment", "Shipment.additionalCharges", "ShipmentAdditionalCharge.isDetention",
		},
		feature: platformcatalog.FeatureDetention,
		answer:  "shipment.additionalCharges.isDetention",
	},
	{
		path:    []string{"Query.shipmentWatchlist", "ShipmentWatchlist.detention"},
		feature: platformcatalog.FeatureDetention,
		answer:  "shipmentWatchlist.detention",
	},
	{
		path:    []string{"Query.invoice", "Invoice.balanceDueMinor"},
		feature: platformcatalog.FeatureAccounting,
		answer:  "invoice.balanceDueMinor",
	},
	{
		path:    []string{"Query.invoice", "Invoice.openBalance"},
		feature: platformcatalog.FeatureAccounting,
		answer:  "invoice.openBalance",
	},
	{
		path:    []string{"Query.invoice", "Invoice.factoringOverride"},
		feature: platformcatalog.FeatureFactoring,
		answer:  "invoice.factoringOverride",
	},
	{
		path:    []string{"Query.invoice", "Invoice.ediSendStatus"},
		feature: platformcatalog.FeatureEDIIntegration,
		answer:  "invoice.ediSendStatus",
	},
	{
		path:    []string{"Query.worker", "Worker.profile", "WorkerProfile.complianceStatus"},
		feature: platformcatalog.FeatureWorkforceCompliance,
		answer:  "worker.profile.complianceStatus",
	},
	{
		path:    []string{"Query.worker", "Worker.profile", "WorkerProfile.safetyRating"},
		feature: platformcatalog.FeatureWorkforceSafety,
		answer:  "worker.profile.safetyRating",
	},
	{
		path:    []string{"Query.customer", "Customer.statusUpdatePreference"},
		feature: platformcatalog.FeatureCustomerTracking,
		answer:  "customer.statusUpdatePreference",
	},
	{
		path: []string{
			"Query.customer", "Customer.billingProfile", "CustomerBillingProfile.creditLimit",
		},
		feature: platformcatalog.FeatureAccounting,
		answer:  "customer.billingProfile.creditLimit",
	},
	{
		path:    []string{"Query.carrier", "Carrier.openIntelEventCount"},
		feature: platformcatalog.FeatureCarrierBrokerage,
		answer:  "carrier.openIntelEventCount",
	},
	{
		path:    []string{"Query.tractor", "Tractor.iftaQualified"},
		feature: platformcatalog.FeatureFuelTax,
		answer:  "tractor.iftaQualified",
	},
}

// On a plan of the base features alone, an add-on's value on a base record is
// answered null without its resolver running, and the response names it with
// its feature.
func TestPlanSchema_ABaseOnlyPlanWithholdsEveryAddOnValueOnABaseRecord(t *testing.T) {
	t.Parallel()

	for _, tc := range addOnValuesOnBaseRecords {
		t.Run(tc.answer, func(t *testing.T) {
			t.Parallel()

			guard, schema, ctx := schemaPlanGuard(t, plantest.BaseOnly())

			result, resolved := interceptSchemaField(t, guard, schema, ctx, tc.path...)

			assert.Nil(t, result)
			assert.False(t, resolved, "the add-on's value is never read")
			assert.Equal(t, []gqlctx.WithheldByPlan{{Path: tc.answer, Feature: tc.feature}},
				withheldByPlan(ctx))
			assert.Empty(t, withheldPaths(ctx), "a plan withholding is not an access withholding")
		})
	}
}

// Without one add-on, only that add-on's values are withheld.
func TestPlanSchema_OnlyTheMissingAddOnsValuesAreWithheld(t *testing.T) {
	t.Parallel()

	guard, schema, ctx := schemaPlanGuard(t, plantest.Without(platformcatalog.FeatureCustomerTracking))

	result, resolved := interceptSchemaField(t, guard, schema, ctx,
		"Query.shipment", "Shipment.tenderStatus")
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)

	result, resolved = interceptSchemaField(t, guard, schema, ctx,
		"Query.shipment", "Shipment.eta")
	assert.Nil(t, result)
	assert.False(t, resolved)

	assert.Equal(t, []gqlctx.WithheldByPlan{{
		Path:    "shipment.eta",
		Feature: platformcatalog.FeatureCustomerTracking,
	}}, withheldByPlan(ctx))
}

// Every plan prices shipments with the standard templates, so a shipment's
// rating method, its inputs and what its last rating billed are answered on a
// base plan, and so is what the tariff told the invoice.
func TestPlanSchema_TheRatingMethodIsAnsweredOnABasePlan(t *testing.T) {
	t.Parallel()

	paths := [][]string{
		{"Query.shipment", "Shipment.formulaTemplateId"},
		{"Query.shipment", "Shipment.tariffInputs"},
		{"Query.shipment", "Shipment.ratingDetail", "ShipmentRatingDetail.bill"},
		{"Query.shipment", "Shipment.ratingDetail", "ShipmentRatingDetail.agreementName"},
	}
	for _, path := range paths {
		t.Run(path[len(path)-1], func(t *testing.T) {
			t.Parallel()

			guard, schema, ctx := schemaPlanGuard(t, plantest.BaseOnly())

			result, resolved := interceptSchemaField(t, guard, schema, ctx, path...)
			assert.Equal(t, "stored value", result)
			assert.True(t, resolved)
			assert.Empty(t, withheldByPlan(ctx))
		})
	}
}

// With every feature on, each value is answered as it was before the plan was
// applied.
func TestPlanSchema_EveryFeatureOnAnswersEveryAddOnValueAsBefore(t *testing.T) {
	t.Parallel()

	for _, tc := range addOnValuesOnBaseRecords {
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

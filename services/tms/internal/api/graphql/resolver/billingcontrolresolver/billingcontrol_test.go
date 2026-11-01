package billingcontrolresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBillingControlResolvers_CheckTheBillingControlPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"billingControl", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).BillingControl(ctx)
			return err
		}},
		{"updateBillingControl", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateBillingControl(
				ctx,
				gqlmodel.UpdateBillingControlInput{},
			)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceBillingControl,
				call.operation,
			)
		})
	}
}

func TestBillingControlFromInput_MapsEveryFieldOntoTheStoredControl(t *testing.T) {
	t.Parallel()

	existing := &billingcontrol.BillingControl{
		ID:             pulid.MustNew("x_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Version:        3,
		CreatedAt:      100,
		UpdatedAt:      200,
	}
	v20 := pulid.MustNew("x_")
	v20Raw := v20.String()
	v24 := "4.5"

	entity, err := billingControlFromInput(existing, &gqlmodel.UpdateBillingControlInput{
		Version:                 3,
		DefaultPaymentTerm:      billingcontrol.PaymentTerm("DueOnReceipt"),
		DefaultInvoiceTerms:     "defaultInvoiceTerms",
		DefaultInvoiceFooter:    "defaultInvoiceFooter",
		ShowDueDateOnInvoice:    true,
		ShowBalanceDueOnInvoice: true,
		ReadyToBillAssignmentMode: billingcontrol.ReadyToBillAssignmentMode(
			"AutomaticWhenEligible",
		),
		BillingQueueTransferMode: billingcontrol.BillingQueueTransferMode(
			"AutomaticWhenReady",
		),
		BillingQueueTransferSchedule:  billingcontrol.TransferSchedule("Weekly"),
		BillingQueueTransferBatchSize: 11,
		InvoiceDraftCreationMode: billingcontrol.InvoiceDraftCreationMode(
			"AutomaticWhenTransferred",
		),
		InvoicePostingMode: billingcontrol.InvoicePostingMode(
			"AutomaticWhenNoBlockingExceptions",
		),
		AutoInvoiceBatchSize:                  14,
		NotifyOnAutoInvoiceCreation:           true,
		ShipmentBillingRequirementEnforcement: tenant.EnforcementLevel("Block"),
		RateValidationEnforcement:             tenant.EnforcementLevel("Block"),
		BillingExceptionDisposition: billingcontrol.BillingExceptionDisposition(
			"ReturnToOperations",
		),
		NotifyOnBillingExceptions:    true,
		RateVarianceTolerancePercent: "12.75",
		RateVarianceAutoResolutionMode: billingcontrol.RateVarianceAutoResolutionMode(
			"BypassReviewWithinTolerance",
		),
		UnratedShipmentDisposition: billingcontrol.UnratedShipmentDisposition("Block"),
		FallbackFormulaTemplateID:  &v20Raw,
		RequireRateOverrideReason:  true,
		EnforceMarginFloor:         true,
		QueueAutoApproveMode: billingcontrol.QueueAutoApproveMode(
			"WhenAgentAgrees",
		),
		QueueAutoApproveMaxAmount: &v24,
		QueueAgentReviewEnabled:   true,
		LateChargeAssessmentMode:  billingcontrol.LateChargeAssessmentMode("Automatic"),
		LateChargeMinimumAmount:   "12.75",
		PaymentRemindersEnabled:   true,
		PaymentReminderDays:       []int{3, 7},
		MonthlyStatementsEnabled:  true,
		MonthlyStatementDay:       34,
	})
	require.NoError(t, err)

	assert.Equal(t, existing.ID, entity.ID)
	assert.Equal(t, existing.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, existing.OrganizationID, entity.OrganizationID)
	assert.Equal(t, int64(100), entity.CreatedAt)
	assert.Equal(t, int64(3), entity.Version)
	assert.Equal(t, billingcontrol.PaymentTerm("DueOnReceipt"), entity.DefaultPaymentTerm)
	assert.Equal(t, "defaultInvoiceTerms", entity.DefaultInvoiceTerms)
	assert.Equal(t, "defaultInvoiceFooter", entity.DefaultInvoiceFooter)
	assert.True(t, entity.ShowDueDateOnInvoice)
	assert.True(t, entity.ShowBalanceDueOnInvoice)
	assert.Equal(
		t,
		billingcontrol.ReadyToBillAssignmentMode("AutomaticWhenEligible"),
		entity.ReadyToBillAssignmentMode,
	)
	assert.Equal(
		t,
		billingcontrol.BillingQueueTransferMode("AutomaticWhenReady"),
		entity.BillingQueueTransferMode,
	)
	assert.Equal(t, billingcontrol.TransferSchedule("Weekly"), entity.BillingQueueTransferSchedule)
	assert.Equal(t, 11, entity.BillingQueueTransferBatchSize)
	assert.Equal(
		t,
		billingcontrol.InvoiceDraftCreationMode("AutomaticWhenTransferred"),
		entity.InvoiceDraftCreationMode,
	)
	assert.Equal(
		t,
		billingcontrol.InvoicePostingMode("AutomaticWhenNoBlockingExceptions"),
		entity.InvoicePostingMode,
	)
	assert.Equal(t, 14, entity.AutoInvoiceBatchSize)
	assert.True(t, entity.NotifyOnAutoInvoiceCreation)
	assert.Equal(t, tenant.EnforcementLevel("Block"), entity.ShipmentBillingRequirementEnforcement)
	assert.Equal(t, tenant.EnforcementLevel("Block"), entity.RateValidationEnforcement)
	assert.Equal(
		t,
		billingcontrol.BillingExceptionDisposition("ReturnToOperations"),
		entity.BillingExceptionDisposition,
	)
	assert.True(t, entity.NotifyOnBillingExceptions)
	assert.Equal(t, "12.75", entity.RateVarianceTolerancePercent.String())
	assert.Equal(
		t,
		billingcontrol.RateVarianceAutoResolutionMode("BypassReviewWithinTolerance"),
		entity.RateVarianceAutoResolutionMode,
	)
	assert.Equal(
		t,
		billingcontrol.UnratedShipmentDisposition("Block"),
		entity.UnratedShipmentDisposition,
	)
	require.NotNil(t, entity.FallbackFormulaTemplateID)
	assert.Equal(t, v20, *entity.FallbackFormulaTemplateID)
	assert.True(t, entity.RequireRateOverrideReason)
	assert.True(t, entity.EnforceMarginFloor)
	assert.Equal(
		t,
		billingcontrol.QueueAutoApproveMode("WhenAgentAgrees"),
		entity.QueueAutoApproveMode,
	)
	assert.True(t, entity.QueueAutoApproveMaxAmount.Valid)
	assert.Equal(t, "4.5", entity.QueueAutoApproveMaxAmount.Decimal.String())
	assert.True(t, entity.QueueAgentReviewEnabled)
	assert.Equal(
		t,
		billingcontrol.LateChargeAssessmentMode("Automatic"),
		entity.LateChargeAssessmentMode,
	)
	assert.Equal(t, "12.75", entity.LateChargeMinimumAmount.String())
	assert.True(t, entity.PaymentRemindersEnabled)
	assert.Equal(t, []int32{3, 7}, entity.PaymentReminderDays)
	assert.True(t, entity.MonthlyStatementsEnabled)
	assert.Equal(t, int16(34), entity.MonthlyStatementDay)
}

func TestBillingControlFromInput_NamesAMalformedField(t *testing.T) {
	t.Parallel()

	_, err := billingControlFromInput(
		&billingcontrol.BillingControl{},
		&gqlmodel.UpdateBillingControlInput{
			RateVarianceTolerancePercent: "twelve",
			LateChargeMinimumAmount:      "1",
		},
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.NotEmpty(t, multiErr.Errors)
	assert.Equal(t, "rateVarianceTolerancePercent", multiErr.Errors[0].Field)
}

func TestBillingControlResolver_RendersDecimalsExactly(t *testing.T) {
	t.Parallel()

	value, err := (&BillingControlResolver{&Deps{}}).RateVarianceTolerancePercent(
		t.Context(),
		&billingcontrol.BillingControl{
			RateVarianceTolerancePercent: decimal.RequireFromString("0.725"),
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "0.725", value)
}

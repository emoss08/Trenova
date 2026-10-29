package billingcontrolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestValidateUpdate_RejectsAutomaticPostingWithoutAutomaticDrafts(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validBillingControl()
	entity.InvoicePostingMode = billingcontrol.InvoicePostingModeAutomaticWhenNoBlockingExceptions
	entity.InvoiceDraftCreationMode = billingcontrol.InvoiceDraftCreationModeManualOnly

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	require.Contains(
		t,
		multiErr.Error(),
		"Automatic invoice posting requires invoice draft creation mode AutomaticWhenTransferred",
	)
}

func TestValidateUpdate_RejectsAutomaticTransferWithoutScheduleAndBatch(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validBillingControl()
	entity.BillingQueueTransferMode = billingcontrol.BillingQueueTransferModeAutomaticWhenReady
	entity.BillingQueueTransferSchedule = ""
	entity.BillingQueueTransferBatchSize = 0

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	require.Contains(t, multiErr.Error(), "Billing queue transfer schedule is required")
	require.Contains(t, multiErr.Error(), "Billing queue transfer batch size must be at least 1")
}

func TestValidateUpdate_AllowsAutomaticDraftAndPostingCombination(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validBillingControl()
	entity.InvoiceDraftCreationMode = billingcontrol.InvoiceDraftCreationModeAutomaticWhenTransferred
	entity.InvoicePostingMode = billingcontrol.InvoicePostingModeAutomaticWhenNoBlockingExceptions
	entity.AutoInvoiceBatchSize = 10

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.Nil(t, multiErr)
}

func validBillingControl() *billingcontrol.BillingControl {
	return &billingcontrol.BillingControl{
		ID:                                    pulid.MustNew("bc_"),
		OrganizationID:                        pulid.MustNew("org_"),
		BusinessUnitID:                        pulid.MustNew("bu_"),
		DefaultPaymentTerm:                    billingcontrol.PaymentTermNet30,
		ReadyToBillAssignmentMode:             billingcontrol.ReadyToBillAssignmentModeManualOnly,
		BillingQueueTransferMode:              billingcontrol.BillingQueueTransferModeManualOnly,
		InvoiceDraftCreationMode:              billingcontrol.InvoiceDraftCreationModeManualOnly,
		InvoicePostingMode:                    billingcontrol.InvoicePostingModeManualReviewRequired,
		ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
		RateValidationEnforcement:             tenant.EnforcementLevelRequireReview,
		BillingExceptionDisposition:           billingcontrol.BillingExceptionDispositionRouteToBillingReview,
		NotifyOnBillingExceptions:             true,
		RateVarianceTolerancePercent:          decimal.RequireFromString("0.010000"),
		RateVarianceAutoResolutionMode:        billingcontrol.RateVarianceAutoResolutionModeDisabled,
	}
}

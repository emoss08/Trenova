package billingcontrol

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func NewDefaultBillingControl(orgID, buID pulid.ID) *BillingControl {
	return &BillingControl{
		OrganizationID:                        orgID,
		BusinessUnitID:                        buID,
		DefaultPaymentTerm:                    PaymentTermNet30,
		ShowDueDateOnInvoice:                  true,
		ShowBalanceDueOnInvoice:               true,
		ReadyToBillAssignmentMode:             ReadyToBillAssignmentModeManualOnly,
		BillingQueueTransferMode:              BillingQueueTransferModeManualOnly,
		BillingQueueTransferSchedule:          TransferScheduleContinuous,
		InvoiceDraftCreationMode:              InvoiceDraftCreationModeManualOnly,
		InvoicePostingMode:                    InvoicePostingModeManualReviewRequired,
		ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
		RateValidationEnforcement:             tenant.EnforcementLevelRequireReview,
		BillingExceptionDisposition:           BillingExceptionDisposition("RouteToBillingReview"),
		NotifyOnBillingExceptions:             true,
		RateVarianceTolerancePercent:          decimal.Zero,
		RateVarianceAutoResolutionMode:        RateVarianceAutoResolutionMode("Disabled"),
		RequireRateOverrideReason:             true,
		UnratedShipmentDisposition: UnratedShipmentDisposition(
			"FallbackFormulaTemplate",
		),
		LateChargeAssessmentMode: LateChargeAssessmentModeDisabled,
		MonthlyStatementDay:      1,
	}
}

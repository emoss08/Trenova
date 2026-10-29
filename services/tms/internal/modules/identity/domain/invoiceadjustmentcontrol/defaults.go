package invoiceadjustmentcontrol

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func NewDefaultInvoiceAdjustmentControl(orgID, buID pulid.ID) *InvoiceAdjustmentControl {
	return &InvoiceAdjustmentControl{
		OrganizationID:                       orgID,
		BusinessUnitID:                       buID,
		PartiallyPaidInvoiceAdjustmentPolicy: AdjustmentEligibilityAllowWithApproval,
		PaidInvoiceAdjustmentPolicy:          AdjustmentEligibilityDisallow,
		DisputedInvoiceAdjustmentPolicy:      AdjustmentEligibilityAllowWithApproval,
		AdjustmentAccountingDatePolicy: AdjustmentAccountingDatePolicy(
			"UseOriginalIfOpenElseNextOpen",
		),
		ClosedPeriodAdjustmentPolicy: ClosedPeriodAdjustmentPolicy(
			"PostInNextOpenPeriodWithApproval",
		),
		AdjustmentReasonRequirement:         RequirementPolicyRequired,
		StandardAdjustmentApprovalPolicy:    ApprovalPolicyAmountThreshold,
		StandardAdjustmentApprovalThreshold: decimal.NewFromFloat(0.01),
		WriteOffApprovalPolicy: WriteOffApprovalPolicy(
			"RequireApprovalAboveThreshold",
		),
		WriteOffApprovalThreshold:      decimal.NewFromFloat(0.01),
		RerateVarianceTolerancePercent: decimal.Zero,
		ReplacementInvoiceReviewPolicy: ReplacementInvoiceReviewPolicy(
			"RequireReviewWhenEconomicTermsChange",
		),
		CustomerCreditBalancePolicy: CustomerCreditBalancePolicy("AllowUnappliedCredit"),
		OverCreditPolicy:            OverCreditPolicyBlock,
		SupersededInvoiceVisibilityPolicy: SupersededInvoiceVisibilityPolicy(
			"ShowCurrentOnlyExternally",
		),
	}
}

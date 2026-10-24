package tenant

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	defaultFunctionalCurrency = "USD"
	defaultUserLocale         = "en"
)

func (u *User) BeforeCreate(ctx context.Context) error {
	if err := u.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil)); err != nil {
		return err
	}
	if u.Status == "" {
		u.Status = domaintypes.StatusActive
	}
	if u.TimeFormat == "" {
		u.TimeFormat = domaintypes.TimeFormat12Hour
	}
	if u.Locale == "" {
		u.Locale = defaultUserLocale
	}
	if u.AvatarStyle == "" {
		u.AvatarStyle = AvatarStyleAuto
	}
	if u.AvailabilityStatus == "" {
		u.AvailabilityStatus = AvailabilityAvailable
	}

	return nil
}

func NewDefaultAccountingControl(orgID, buID pulid.ID) *AccountingControl {
	return &AccountingControl{
		OrganizationID:                  orgID,
		BusinessUnitID:                  buID,
		AccountingBasis:                 AccountingBasisAccrual,
		RevenueRecognitionPolicy:        RevenueRecognitionOnInvoicePost,
		ExpenseRecognitionPolicy:        ExpenseRecognitionOnVendorBillPost,
		JournalPostingMode:              JournalPostingModeManual,
		ManualJournalEntryPolicy:        ManualJournalEntryPolicyAdjustmentOnly,
		RequireManualJEApproval:         true,
		JournalReversalPolicy:           JournalReversalPolicyNextOpenPeriod,
		PeriodCloseMode:                 PeriodCloseModeManualOnly,
		RequirePeriodCloseApproval:      true,
		LockedPeriodPostingPolicy:       LockedPeriodPostingPolicy("BlockSubledgerAllowManualJe"),
		ClosedPeriodPostingPolicy:       ClosedPeriodPostingPolicyRequireReopen,
		ReconciliationMode:              ReconciliationModeDisabled,
		ReconciliationToleranceAmount:   decimal.Zero,
		NotifyOnReconciliationException: true,
		CurrencyMode:                    CurrencyModeSingleCurrency,
		FunctionalCurrencyCode:          defaultFunctionalCurrency,
		ExchangeRateDatePolicy:          ExchangeRateDatePolicyDocumentDate,
		ExchangeRateOverridePolicy:      ExchangeRateOverrideRequireApproval,
	}
}

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
		ShipmentBillingRequirementEnforcement: EnforcementLevelBlock,
		RateValidationEnforcement:             EnforcementLevelRequireReview,
		BillingExceptionDisposition:           BillingExceptionDisposition("RouteToBillingReview"),
		NotifyOnBillingExceptions:             true,
		RateVarianceTolerancePercent:          decimal.Zero,
		RateVarianceAutoResolutionMode:        RateVarianceAutoResolutionMode("Disabled"),
		RequireRateOverrideReason:             true,
		UnratedShipmentDisposition:            UnratedShipmentDisposition("FallbackFormulaTemplate"),
		LateChargeAssessmentMode:              LateChargeAssessmentModeDisabled,
		MonthlyStatementDay:                   1,
	}
}

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
		WriteOffApprovalPolicy:              WriteOffApprovalPolicy("RequireApprovalAboveThreshold"),
		WriteOffApprovalThreshold:           decimal.NewFromFloat(0.01),
		RerateVarianceTolerancePercent:      decimal.Zero,
		ReplacementInvoiceReviewPolicy: ReplacementInvoiceReviewPolicy(
			"RequireReviewWhenEconomicTermsChange",
		),
		CustomerCreditBalancePolicy:       CustomerCreditBalancePolicy("AllowUnappliedCredit"),
		OverCreditPolicy:                  OverCreditPolicyBlock,
		SupersededInvoiceVisibilityPolicy: SupersededInvoiceVisibilityPolicy("ShowCurrentOnlyExternally"),
	}
}

func NewDefaultShipmentControl(orgID, buID pulid.ID) *ShipmentControl {
	threshold := int16(30)

	return &ShipmentControl{
		OrganizationID:               orgID,
		BusinessUnitID:               buID,
		MaxShipmentWeightLimit:       80000,
		AutoDelayShipments:           true,
		AutoDelayShipmentsThreshold:  &threshold,
		DetentionThreshold:           new(threshold),
		AutoCancelShipmentsThreshold: new(threshold),
		AllowMoveRemovals:            true,
		CheckForDuplicateBOLs:        true,
		CheckHazmatSegregation:       true,
	}
}

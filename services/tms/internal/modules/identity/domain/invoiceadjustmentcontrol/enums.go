package invoiceadjustmentcontrol

type AdjustmentEligibilityPolicy string

const (
	AdjustmentEligibilityDisallow             = AdjustmentEligibilityPolicy("Disallow")
	AdjustmentEligibilityAllowWithApproval    = AdjustmentEligibilityPolicy("AllowWithApproval")
	AdjustmentEligibilityAllowWithoutApproval = AdjustmentEligibilityPolicy("AllowWithoutApproval")
)

type AdjustmentAccountingDatePolicy string

const (
	AdjustmentAccountingDateUseOriginalIfOpenElseNextOpen = AdjustmentAccountingDatePolicy(
		"UseOriginalIfOpenElseNextOpen",
	)
	AdjustmentAccountingDateAlwaysNextOpen = AdjustmentAccountingDatePolicy(
		"AlwaysNextOpen",
	)
)

type ClosedPeriodAdjustmentPolicy string

const (
	ClosedPeriodAdjustmentPolicyDisallow = ClosedPeriodAdjustmentPolicy(
		"Disallow",
	)
	ClosedPeriodAdjustmentPolicyRequireReopen = ClosedPeriodAdjustmentPolicy(
		"RequireReopen",
	)
	ClosedPeriodAdjustmentPolicyPostInNextOpenPeriodWithApproval = ClosedPeriodAdjustmentPolicy(
		"PostInNextOpenPeriodWithApproval",
	)
)

type RequirementPolicy string

const (
	RequirementPolicyOptional = RequirementPolicy("Optional")
	RequirementPolicyRequired = RequirementPolicy("Required")
)

type ApprovalPolicy string

const (
	ApprovalPolicyNone            = ApprovalPolicy("None")
	ApprovalPolicyAlways          = ApprovalPolicy("Always")
	ApprovalPolicyAmountThreshold = ApprovalPolicy("AmountThreshold")
)

type WriteOffApprovalPolicy string

const (
	WriteOffApprovalPolicyDisallow              = WriteOffApprovalPolicy("Disallow")
	WriteOffApprovalPolicyAlwaysRequireApproval = WriteOffApprovalPolicy(
		"AlwaysRequireApproval",
	)
	WriteOffApprovalPolicyRequireApprovalAboveThreshold = WriteOffApprovalPolicy(
		"RequireApprovalAboveThreshold",
	)
)

type ReplacementInvoiceReviewPolicy string

const (
	ReplacementInvoiceReviewPolicyNoAdditionalReview = ReplacementInvoiceReviewPolicy(
		"NoAdditionalReview",
	)
	ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange = ReplacementInvoiceReviewPolicy(
		"RequireReviewWhenEconomicTermsChange",
	)
	ReplacementInvoiceReviewPolicyAlwaysRequireReview = ReplacementInvoiceReviewPolicy(
		"AlwaysRequireReview",
	)
)

type CustomerCreditBalancePolicy string

const (
	CustomerCreditBalancePolicyDisallow             = CustomerCreditBalancePolicy("Disallow")
	CustomerCreditBalancePolicyAllowUnappliedCredit = CustomerCreditBalancePolicy(
		"AllowUnappliedCredit",
	)
)

type OverCreditPolicy string

const (
	OverCreditPolicyBlock             = OverCreditPolicy("Block")
	OverCreditPolicyAllowWithApproval = OverCreditPolicy("AllowWithApproval")
)

type SupersededInvoiceVisibilityPolicy string

const (
	SupersededInvoiceVisibilityPolicyShowCurrentOnlyExternally = SupersededInvoiceVisibilityPolicy(
		"ShowCurrentOnlyExternally",
	)
	SupersededInvoiceVisibilityPolicyShowCurrentAndSupersededExternally = SupersededInvoiceVisibilityPolicy(
		"ShowCurrentAndSupersededExternally",
	)
)

func (v AdjustmentEligibilityPolicy) IsValid() bool {
	switch v {
	case AdjustmentEligibilityDisallow,
		AdjustmentEligibilityAllowWithApproval,
		AdjustmentEligibilityAllowWithoutApproval:
		return true
	default:
		return false
	}
}

func (v AdjustmentAccountingDatePolicy) IsValid() bool {
	switch v {
	case AdjustmentAccountingDateUseOriginalIfOpenElseNextOpen,
		AdjustmentAccountingDateAlwaysNextOpen:
		return true
	default:
		return false
	}
}

func (v ClosedPeriodAdjustmentPolicy) IsValid() bool {
	switch v {
	case ClosedPeriodAdjustmentPolicyDisallow,
		ClosedPeriodAdjustmentPolicyRequireReopen,
		ClosedPeriodAdjustmentPolicyPostInNextOpenPeriodWithApproval:
		return true
	default:
		return false
	}
}

func (v RequirementPolicy) IsValid() bool {
	switch v {
	case RequirementPolicyOptional,
		RequirementPolicyRequired:
		return true
	default:
		return false
	}
}

func (v ApprovalPolicy) IsValid() bool {
	switch v {
	case ApprovalPolicyNone,
		ApprovalPolicyAlways,
		ApprovalPolicyAmountThreshold:
		return true
	default:
		return false
	}
}

func (v WriteOffApprovalPolicy) IsValid() bool {
	switch v {
	case WriteOffApprovalPolicyDisallow,
		WriteOffApprovalPolicyAlwaysRequireApproval,
		WriteOffApprovalPolicyRequireApprovalAboveThreshold:
		return true
	default:
		return false
	}
}

func (v ReplacementInvoiceReviewPolicy) IsValid() bool {
	switch v {
	case ReplacementInvoiceReviewPolicyNoAdditionalReview,
		ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange,
		ReplacementInvoiceReviewPolicyAlwaysRequireReview:
		return true
	default:
		return false
	}
}

func (v CustomerCreditBalancePolicy) IsValid() bool {
	switch v {
	case CustomerCreditBalancePolicyDisallow,
		CustomerCreditBalancePolicyAllowUnappliedCredit:
		return true
	default:
		return false
	}
}

func (v OverCreditPolicy) IsValid() bool {
	switch v {
	case OverCreditPolicyBlock,
		OverCreditPolicyAllowWithApproval:
		return true
	default:
		return false
	}
}

func (v SupersededInvoiceVisibilityPolicy) IsValid() bool {
	switch v {
	case SupersededInvoiceVisibilityPolicyShowCurrentOnlyExternally,
		SupersededInvoiceVisibilityPolicyShowCurrentAndSupersededExternally:
		return true
	default:
		return false
	}
}

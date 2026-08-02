package customer

type BillingCycleType string

const (
	BillingCycleTypeImmediate = BillingCycleType("Immediate")
	BillingCycleTypeDaily     = BillingCycleType("Daily")
	BillingCycleTypeWeekly    = BillingCycleType("Weekly")
	BillingCycleTypeBiWeekly  = BillingCycleType("BiWeekly")

	BillingCycleTypeMonthly = BillingCycleType("Monthly")

	BillingCycleTypeQuarterly   = BillingCycleType("Quarterly")
	BillingCycleTypePerShipment = BillingCycleType("PerShipment")
)

type PaymentTerm string

const (
	PaymentTermNet10        = PaymentTerm("Net10")
	PaymentTermNet15        = PaymentTerm("Net15")
	PaymentTermNet30        = PaymentTerm("Net30")
	PaymentTermNet45        = PaymentTerm("Net45")
	PaymentTermNet60        = PaymentTerm("Net60")
	PaymentTermNet90        = PaymentTerm("Net90")
	PaymentTermDueOnReceipt = PaymentTerm("DueOnReceipt")
)

type CreditStatus string

const (
	CreditStatusActive    = CreditStatus("Active")
	CreditStatusWarning   = CreditStatus("Warning")
	CreditStatusHold      = CreditStatus("Hold")
	CreditStatusSuspended = CreditStatus("Suspended")
	CreditStatusReview    = CreditStatus("Review")
)

type FuelSurchargeMode string

const (
	FuelSurchargeModeNone         = FuelSurchargeMode("None")
	FuelSurchargeModeProgram      = FuelSurchargeMode("Program")
	FuelSurchargeModeFuelIncluded = FuelSurchargeMode("FuelIncluded")
)

type InvoiceNumberFormat string

const (
	InvoiceNumberFormatDefault      = InvoiceNumberFormat("Default")
	InvoiceNumberFormatCustomPrefix = InvoiceNumberFormat("CustomPrefix")
	InvoiceNumberFormatPOBased      = InvoiceNumberFormat("POBased")
)

type ConsolidationGroupBy string

const (
	ConsolidationGroupByNone     = ConsolidationGroupBy("None")
	ConsolidationGroupByLocation = ConsolidationGroupBy("Location")
	ConsolidationGroupByPONumber = ConsolidationGroupBy("PONumber")
	ConsolidationGroupByBOL      = ConsolidationGroupBy("BOL")
	ConsolidationGroupByDivision = ConsolidationGroupBy("Division")
)

type InvoiceMethod string

const (
	InvoiceMethodIndividual        = InvoiceMethod("Individual")
	InvoiceMethodSummary           = InvoiceMethod("Summary")
	InvoiceMethodSummaryWithDetail = InvoiceMethod("SummaryWithDetail")
)

type InvoiceAdjustmentSupportingDocumentPolicy string

const (
	InvoiceAdjustmentSupportingDocumentPolicyInherit = InvoiceAdjustmentSupportingDocumentPolicy(
		"Inherit",
	)
	InvoiceAdjustmentSupportingDocumentPolicyRequired = InvoiceAdjustmentSupportingDocumentPolicy(
		"Required",
	)
	InvoiceAdjustmentSupportingDocumentPolicyOptional = InvoiceAdjustmentSupportingDocumentPolicy(
		"Optional",
	)
)

func (t BillingCycleType) IsValid() bool {
	switch t {
	case BillingCycleTypeImmediate, BillingCycleTypeDaily, BillingCycleTypeWeekly,
		BillingCycleTypeBiWeekly, BillingCycleTypeMonthly, BillingCycleTypeQuarterly,
		BillingCycleTypePerShipment:
		return true
	default:
		return false
	}
}

func (t PaymentTerm) IsValid() bool {
	switch t {
	case PaymentTermNet10, PaymentTermNet15, PaymentTermNet30, PaymentTermNet45,
		PaymentTermNet60, PaymentTermNet90, PaymentTermDueOnReceipt:
		return true
	default:
		return false
	}
}

func (s CreditStatus) IsValid() bool {
	switch s {
	case CreditStatusActive, CreditStatusWarning, CreditStatusHold,
		CreditStatusSuspended, CreditStatusReview:
		return true
	default:
		return false
	}
}

func (m FuelSurchargeMode) IsValid() bool {
	switch m {
	case FuelSurchargeModeNone, FuelSurchargeModeProgram, FuelSurchargeModeFuelIncluded:
		return true
	default:
		return false
	}
}

func (f InvoiceNumberFormat) IsValid() bool {
	switch f {
	case InvoiceNumberFormatDefault, InvoiceNumberFormatCustomPrefix, InvoiceNumberFormatPOBased:
		return true
	default:
		return false
	}
}

func (g ConsolidationGroupBy) IsValid() bool {
	switch g {
	case ConsolidationGroupByNone, ConsolidationGroupByLocation, ConsolidationGroupByPONumber,
		ConsolidationGroupByBOL, ConsolidationGroupByDivision:
		return true
	default:
		return false
	}
}

func (m InvoiceMethod) IsValid() bool {
	switch m {
	case InvoiceMethodIndividual, InvoiceMethodSummary, InvoiceMethodSummaryWithDetail:
		return true
	default:
		return false
	}
}

func (p InvoiceAdjustmentSupportingDocumentPolicy) IsValid() bool {
	switch p {
	case InvoiceAdjustmentSupportingDocumentPolicyInherit,
		InvoiceAdjustmentSupportingDocumentPolicyRequired,
		InvoiceAdjustmentSupportingDocumentPolicyOptional:
		return true
	default:
		return false
	}
}

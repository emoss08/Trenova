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

type FuelSurchargeMethod string

const (
	FuelSurchargeMethodNone         = FuelSurchargeMethod("None")
	FuelSurchargeMethodPercentage   = FuelSurchargeMethod("Percentage")
	FuelSurchargeMethodPerMile      = FuelSurchargeMethod("PerMile")
	FuelSurchargeMethodFlatSchedule = FuelSurchargeMethod("FlatSchedule")
	FuelSurchargeMethodIncluded     = FuelSurchargeMethod("Included")
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

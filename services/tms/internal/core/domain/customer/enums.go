package customer

// InvoiceDelivery is how many invoices a customer's freight turns into:
// one each, one per order, or one per billing period.
type InvoiceDelivery string

const (
	InvoiceDeliveryPerShipment  = InvoiceDelivery("PerShipment")
	InvoiceDeliveryPerOrder     = InvoiceDelivery("PerOrder")
	InvoiceDeliveryConsolidated = InvoiceDelivery("Consolidated")
)

// BillingCycle is how often a consolidated customer is billed. It answers only
// the cadence question; what the period is split into is SplitBy.
type BillingCycle string

const (
	BillingCycleImmediate   = BillingCycle("Immediate")
	BillingCycleDaily       = BillingCycle("Daily")
	BillingCycleWeekly      = BillingCycle("Weekly")
	BillingCycleBiWeekly    = BillingCycle("BiWeekly")
	BillingCycleSemiMonthly = BillingCycle("SemiMonthly")
	BillingCycleMonthly     = BillingCycle("Monthly")
	BillingCycleQuarterly   = BillingCycle("Quarterly")
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

// InvoiceSplitKey decides how many invoices a billing period yields. Customer
// means one; every other member means one per distinct value of that key.
//
// There is deliberately no Division member: no division entity exists anywhere
// in the domain, and a shipment carries no fleet code to stand in for one.
type InvoiceSplitKey string

const (
	InvoiceSplitKeyCustomer               = InvoiceSplitKey("Customer")
	InvoiceSplitKeyCustomerAndPONumber    = InvoiceSplitKey("CustomerAndPONumber")
	InvoiceSplitKeyCustomerAndShipmentBOL = InvoiceSplitKey("CustomerAndShipmentBOL")
	InvoiceSplitKeyCustomerAndOrder       = InvoiceSplitKey("CustomerAndOrder")
	InvoiceSplitKeyCustomerAndOrigin      = InvoiceSplitKey("CustomerAndOrigin")
	InvoiceSplitKeyCustomerAndDestination = InvoiceSplitKey("CustomerAndDestination")
	InvoiceSplitKeyCustomerAndServiceType = InvoiceSplitKey("CustomerAndServiceType")
)

// InvoiceSectionKey decides how the lines inside one invoice are organised. It
// never changes how many invoices there are.
type InvoiceSectionKey string

const (
	InvoiceSectionKeyShipment    = InvoiceSectionKey("Shipment")
	InvoiceSectionKeyPONumber    = InvoiceSectionKey("PONumber")
	InvoiceSectionKeyOrigin      = InvoiceSectionKey("Origin")
	InvoiceSectionKeyDestination = InvoiceSectionKey("Destination")
)

// InvoiceDetail is how verbose each section is. It replaces the rendering half
// of the old InvoiceMethod, whose other half was a delivery mode.
type InvoiceDetail string

const (
	InvoiceDetailDetailed = InvoiceDetail("Detailed")
	InvoiceDetailSummary  = InvoiceDetail("Summary")
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

func (d InvoiceDelivery) IsValid() bool {
	switch d {
	case InvoiceDeliveryPerShipment, InvoiceDeliveryPerOrder, InvoiceDeliveryConsolidated:
		return true
	default:
		return false
	}
}

func (c BillingCycle) IsValid() bool {
	switch c {
	case BillingCycleImmediate, BillingCycleDaily, BillingCycleWeekly,
		BillingCycleBiWeekly, BillingCycleSemiMonthly, BillingCycleMonthly,
		BillingCycleQuarterly:
		return true
	default:
		return false
	}
}

// IsPeriodic reports whether the cycle closes a period that has to be swept,
// rather than billing the moment a shipment is approved.
func (c BillingCycle) IsPeriodic() bool {
	return c != "" && c != BillingCycleImmediate
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

func (k InvoiceSplitKey) IsValid() bool {
	switch k {
	case InvoiceSplitKeyCustomer, InvoiceSplitKeyCustomerAndPONumber,
		InvoiceSplitKeyCustomerAndShipmentBOL, InvoiceSplitKeyCustomerAndOrder,
		InvoiceSplitKeyCustomerAndOrigin, InvoiceSplitKeyCustomerAndDestination,
		InvoiceSplitKeyCustomerAndServiceType:
		return true
	default:
		return false
	}
}

func (k InvoiceSectionKey) IsValid() bool {
	switch k {
	case InvoiceSectionKeyShipment, InvoiceSectionKeyPONumber,
		InvoiceSectionKeyOrigin, InvoiceSectionKeyDestination:
		return true
	default:
		return false
	}
}

func (d InvoiceDetail) IsValid() bool {
	switch d {
	case InvoiceDetailDetailed, InvoiceDetailSummary:
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

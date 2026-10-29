package billingcontrol

type TransferSchedule string

const (
	TransferScheduleContinuous = TransferSchedule("Continuous")
	TransferScheduleHourly     = TransferSchedule("Hourly")
	TransferScheduleDaily      = TransferSchedule("Daily")
	TransferScheduleWeekly     = TransferSchedule("Weekly")
)

type BillingExceptionDisposition string

const (
	BillingExceptionDispositionRouteToBillingReview = BillingExceptionDisposition(
		"RouteToBillingReview",
	)
	BillingExceptionDispositionReturnToOperations = BillingExceptionDisposition(
		"ReturnToOperations",
	)
)

type ReadyToBillAssignmentMode string

const (
	ReadyToBillAssignmentModeManualOnly            = ReadyToBillAssignmentMode("ManualOnly")
	ReadyToBillAssignmentModeAutomaticWhenEligible = ReadyToBillAssignmentMode(
		"AutomaticWhenEligible",
	)
)

type BillingQueueTransferMode string

const (
	BillingQueueTransferModeManualOnly         = BillingQueueTransferMode("ManualOnly")
	BillingQueueTransferModeAutomaticWhenReady = BillingQueueTransferMode("AutomaticWhenReady")
)

type InvoiceDraftCreationMode string

const (
	InvoiceDraftCreationModeManualOnly               = InvoiceDraftCreationMode("ManualOnly")
	InvoiceDraftCreationModeAutomaticWhenTransferred = InvoiceDraftCreationMode(
		"AutomaticWhenTransferred",
	)
)

type InvoicePostingMode string

const (
	InvoicePostingModeManualReviewRequired              = InvoicePostingMode("ManualReviewRequired")
	InvoicePostingModeAutomaticWhenNoBlockingExceptions = InvoicePostingMode(
		"AutomaticWhenNoBlockingExceptions",
	)
)

// LateChargeAssessmentMode says what the nightly late-charge run may do.
type LateChargeAssessmentMode string

const (
	LateChargeAssessmentModeDisabled  = LateChargeAssessmentMode("Disabled")
	LateChargeAssessmentModePreview   = LateChargeAssessmentMode("Preview")
	LateChargeAssessmentModeAutomatic = LateChargeAssessmentMode("Automatic")
)

func (m LateChargeAssessmentMode) IsValid() bool {
	switch m {
	case LateChargeAssessmentModeDisabled,
		LateChargeAssessmentModePreview,
		LateChargeAssessmentModeAutomatic:
		return true
	default:
		return false
	}
}

type RateVarianceAutoResolutionMode string

const (
	RateVarianceAutoResolutionModeDisabled = RateVarianceAutoResolutionMode(
		"Disabled",
	)
	RateVarianceAutoResolutionModeBypassReviewWithinTolerance = RateVarianceAutoResolutionMode(
		"BypassReviewWithinTolerance",
	)
)

// UnratedShipmentDisposition says what happens when no rate agreement covers a
// shipment's lane.
//
// The default preserves the behaviour that existed before agreements: fall back
// to the formula template on the shipment. An organization that has not written
// a contract yet therefore sees no change whatsoever, and opting in is a matter
// of writing the first agreement rather than flipping a switch.
type UnratedShipmentDisposition string

const (
	// UnratedShipmentDispositionFallbackFormulaTemplate prices from the
	// shipment's own formula template, or the organization's fallback template.
	UnratedShipmentDispositionFallbackFormulaTemplate = UnratedShipmentDisposition(
		"FallbackFormulaTemplate",
	)
	// UnratedShipmentDispositionZeroAndFlag prices at nothing and raises a
	// billing exception, for organizations that would rather see the gap than
	// have a number invented for it.
	UnratedShipmentDispositionZeroAndFlag = UnratedShipmentDisposition("ZeroAndFlag")
	// UnratedShipmentDispositionBlock refuses the save outright.
	UnratedShipmentDispositionBlock = UnratedShipmentDisposition("Block")
)

func (usd UnratedShipmentDisposition) String() string {
	return string(usd)
}

func (usd UnratedShipmentDisposition) IsValid() bool {
	switch usd {
	case UnratedShipmentDispositionFallbackFormulaTemplate,
		UnratedShipmentDispositionZeroAndFlag,
		UnratedShipmentDispositionBlock:
		return true
	default:
		return false
	}
}

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

func (v TransferSchedule) IsValid() bool {
	switch v {
	case TransferScheduleContinuous,
		TransferScheduleHourly,
		TransferScheduleDaily,
		TransferScheduleWeekly:
		return true
	default:
		return false
	}
}

func (v BillingExceptionDisposition) IsValid() bool {
	switch v {
	case BillingExceptionDispositionRouteToBillingReview,
		BillingExceptionDispositionReturnToOperations:
		return true
	default:
		return false
	}
}

func (v ReadyToBillAssignmentMode) IsValid() bool {
	switch v {
	case ReadyToBillAssignmentModeManualOnly,
		ReadyToBillAssignmentModeAutomaticWhenEligible:
		return true
	default:
		return false
	}
}

func (v BillingQueueTransferMode) IsValid() bool {
	switch v {
	case BillingQueueTransferModeManualOnly,
		BillingQueueTransferModeAutomaticWhenReady:
		return true
	default:
		return false
	}
}

func (v InvoiceDraftCreationMode) IsValid() bool {
	switch v {
	case InvoiceDraftCreationModeManualOnly,
		InvoiceDraftCreationModeAutomaticWhenTransferred:
		return true
	default:
		return false
	}
}

func (v InvoicePostingMode) IsValid() bool {
	switch v {
	case InvoicePostingModeManualReviewRequired,
		InvoicePostingModeAutomaticWhenNoBlockingExceptions:
		return true
	default:
		return false
	}
}

func (v RateVarianceAutoResolutionMode) IsValid() bool {
	switch v {
	case RateVarianceAutoResolutionModeDisabled,
		RateVarianceAutoResolutionModeBypassReviewWithinTolerance:
		return true
	default:
		return false
	}
}

func (v PaymentTerm) IsValid() bool {
	switch v {
	case PaymentTermNet10,
		PaymentTermNet15,
		PaymentTermNet30,
		PaymentTermNet45,
		PaymentTermNet60,
		PaymentTermNet90,
		PaymentTermDueOnReceipt:
		return true
	default:
		return false
	}
}

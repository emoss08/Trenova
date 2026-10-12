package driverpay

type PayeeClassification string

const (
	PayeeClassificationCompanyDriver = PayeeClassification("CompanyDriver")
	PayeeClassificationOwnerOperator = PayeeClassification("OwnerOperator")
)

func (p PayeeClassification) String() string { return string(p) }

func (p PayeeClassification) IsValid() bool {
	switch p {
	case PayeeClassificationCompanyDriver, PayeeClassificationOwnerOperator:
		return true
	default:
		return false
	}
}

type ComponentKind string

const (
	ComponentKindLinehaul      = ComponentKind("Linehaul")
	ComponentKindFuelSurcharge = ComponentKind("FuelSurcharge")
	ComponentKindStopPay       = ComponentKind("StopPay")
	ComponentKindDetention     = ComponentKind("Detention")
	ComponentKindLayover       = ComponentKind("Layover")
	ComponentKindBreakdown     = ComponentKind("Breakdown")
	ComponentKindTarp          = ComponentKind("Tarp")
	ComponentKindHazmat        = ComponentKind("Hazmat")
	ComponentKindBonus         = ComponentKind("Bonus")
	ComponentKindCustom        = ComponentKind("Custom")
)

func (c ComponentKind) String() string { return string(c) }

func (c ComponentKind) IsValid() bool {
	switch c {
	case ComponentKindLinehaul, ComponentKindFuelSurcharge, ComponentKindStopPay,
		ComponentKindDetention, ComponentKindLayover, ComponentKindBreakdown,
		ComponentKindTarp, ComponentKindHazmat, ComponentKindBonus, ComponentKindCustom:
		return true
	default:
		return false
	}
}

type CalcMethod string

const (
	CalcMethodPerLoadedMile    = CalcMethod("PerLoadedMile")
	CalcMethodPerEmptyMile     = CalcMethod("PerEmptyMile")
	CalcMethodPerTotalMile     = CalcMethod("PerTotalMile")
	CalcMethodPercentOfRevenue = CalcMethod("PercentOfRevenue")
	CalcMethodFlatPerShipment  = CalcMethod("FlatPerShipment")
	CalcMethodPerStop          = CalcMethod("PerStop")
	CalcMethodPerHour          = CalcMethod("PerHour")
	CalcMethodPerDay           = CalcMethod("PerDay")
	CalcMethodPerEvent         = CalcMethod("PerEvent")
)

func (c CalcMethod) String() string { return string(c) }

func (c CalcMethod) IsValid() bool {
	switch c {
	case CalcMethodPerLoadedMile, CalcMethodPerEmptyMile, CalcMethodPerTotalMile,
		CalcMethodPercentOfRevenue, CalcMethodFlatPerShipment, CalcMethodPerStop,
		CalcMethodPerHour, CalcMethodPerDay, CalcMethodPerEvent:
		return true
	default:
		return false
	}
}

func (c CalcMethod) IsPerMile() bool {
	switch c { //nolint:exhaustive // only per-mile methods matter; default covers the rest
	case CalcMethodPerLoadedMile, CalcMethodPerEmptyMile, CalcMethodPerTotalMile:
		return true
	default:
		return false
	}
}

type RevenueBasis string

const (
	RevenueBasisLinehaul                  = RevenueBasis("Linehaul")
	RevenueBasisLinehaulPlusFuelSurcharge = RevenueBasis("LinehaulPlusFuelSurcharge")
	RevenueBasisTotalRevenue              = RevenueBasis("TotalRevenue")
)

func (r RevenueBasis) String() string { return string(r) }

func (r RevenueBasis) IsValid() bool {
	switch r {
	case RevenueBasisLinehaul, RevenueBasisLinehaulPlusFuelSurcharge, RevenueBasisTotalRevenue:
		return true
	default:
		return false
	}
}

type EarningFrequency string

const (
	EarningFrequencyEverySettlement = EarningFrequency("EverySettlement")
	EarningFrequencyMonthly         = EarningFrequency("Monthly")
)

func (e EarningFrequency) String() string { return string(e) }

func (e EarningFrequency) IsValid() bool {
	switch e {
	case EarningFrequencyEverySettlement, EarningFrequencyMonthly:
		return true
	default:
		return false
	}
}

type EarningStatus string

const (
	EarningStatusActive    = EarningStatus("Active")
	EarningStatusPaused    = EarningStatus("Paused")
	EarningStatusCompleted = EarningStatus("Completed")
)

func (e EarningStatus) String() string { return string(e) }

func (e EarningStatus) IsValid() bool {
	switch e {
	case EarningStatusActive, EarningStatusPaused, EarningStatusCompleted:
		return true
	default:
		return false
	}
}

type DeductionFrequency string

const (
	DeductionFrequencyEverySettlement = DeductionFrequency("EverySettlement")
	DeductionFrequencyMonthly         = DeductionFrequency("Monthly")
)

func (d DeductionFrequency) String() string { return string(d) }

func (d DeductionFrequency) IsValid() bool {
	switch d {
	case DeductionFrequencyEverySettlement, DeductionFrequencyMonthly:
		return true
	default:
		return false
	}
}

// DeductionKind separates the three reasons money comes off a settlement.
// They travel down the same path — the settlement builder does not care why —
// but a garnishment carries a court order and a benefit carries an enrollment,
// and neither can be told from an ordinary deduction without this.
type DeductionKind string

const (
	DeductionKindStandard    = DeductionKind("Standard")
	DeductionKindGarnishment = DeductionKind("Garnishment")
	DeductionKindBenefit     = DeductionKind("Benefit")
)

func (d DeductionKind) String() string { return string(d) }

func (d DeductionKind) IsValid() bool {
	switch d {
	case DeductionKindStandard, DeductionKindGarnishment, DeductionKindBenefit:
		return true
	default:
		return false
	}
}

// DefaultPriority is the order deductions are taken in when pay will not cover
// them all. A court order outranks a voluntary deduction, so a garnishment
// sorts first; a benefit a worker chose sorts last.
func (d DeductionKind) DefaultPriority() int16 {
	switch d {
	case DeductionKindGarnishment:
		return 10
	case DeductionKindBenefit:
		return 200
	default:
		return 100
	}
}

// EarningKind separates ongoing pay from a one-off. A bonus is an ordinary
// earning whose cap equals its amount, so it pays once and completes itself
// through the same path everything else uses; the kind is what lets a screen
// offer it as a bonus rather than making somebody set the cap by hand.
type EarningKind string

const (
	EarningKindStandard = EarningKind("Standard")
	EarningKindBonus    = EarningKind("Bonus")
)

func (e EarningKind) String() string { return string(e) }

func (e EarningKind) IsValid() bool {
	switch e {
	case EarningKindStandard, EarningKindBonus:
		return true
	default:
		return false
	}
}

type DeductionStatus string

const (
	DeductionStatusActive    = DeductionStatus("Active")
	DeductionStatusPaused    = DeductionStatus("Paused")
	DeductionStatusCompleted = DeductionStatus("Completed")
)

func (d DeductionStatus) String() string { return string(d) }

func (d DeductionStatus) IsValid() bool {
	switch d {
	case DeductionStatusActive, DeductionStatusPaused, DeductionStatusCompleted:
		return true
	default:
		return false
	}
}

type AdvanceStatus string

const (
	AdvanceStatusOutstanding        = AdvanceStatus("Outstanding")
	AdvanceStatusPartiallyRecovered = AdvanceStatus("PartiallyRecovered")
	AdvanceStatusRecovered          = AdvanceStatus("Recovered")
	AdvanceStatusWrittenOff         = AdvanceStatus("WrittenOff")
)

func (a AdvanceStatus) String() string { return string(a) }

func (a AdvanceStatus) IsValid() bool {
	switch a {
	case AdvanceStatusOutstanding, AdvanceStatusPartiallyRecovered,
		AdvanceStatusRecovered, AdvanceStatusWrittenOff:
		return true
	default:
		return false
	}
}

type AdvanceSource string

const (
	AdvanceSourceCash         = AdvanceSource("Cash")
	AdvanceSourceEFSMoneyCode = AdvanceSource("EFSMoneyCode")
	AdvanceSourceComdataCode  = AdvanceSource("ComdataCode")
	AdvanceSourceFuelCard     = AdvanceSource("FuelCard")
	AdvanceSourceOther        = AdvanceSource("Other")
)

func (a AdvanceSource) String() string { return string(a) }

func (a AdvanceSource) IsValid() bool {
	switch a {
	case AdvanceSourceCash, AdvanceSourceEFSMoneyCode, AdvanceSourceComdataCode,
		AdvanceSourceFuelCard, AdvanceSourceOther:
		return true
	default:
		return false
	}
}

type EscrowAccountStatus string

const (
	EscrowAccountStatusActive = EscrowAccountStatus("Active")
	EscrowAccountStatusClosed = EscrowAccountStatus("Closed")
)

func (e EscrowAccountStatus) String() string { return string(e) }

func (e EscrowAccountStatus) IsValid() bool {
	switch e {
	case EscrowAccountStatusActive, EscrowAccountStatusClosed:
		return true
	default:
		return false
	}
}

type EscrowTransactionType string

const (
	EscrowTransactionTypeContribution    = EscrowTransactionType("Contribution")
	EscrowTransactionTypeInterestAccrual = EscrowTransactionType("InterestAccrual")
	EscrowTransactionTypeApplication     = EscrowTransactionType("Application")
	EscrowTransactionTypeRefund          = EscrowTransactionType("Refund")
	EscrowTransactionTypeAdjustment      = EscrowTransactionType("Adjustment")
)

func (e EscrowTransactionType) String() string { return string(e) }

func (e EscrowTransactionType) IsValid() bool {
	switch e {
	case EscrowTransactionTypeContribution, EscrowTransactionTypeInterestAccrual,
		EscrowTransactionTypeApplication, EscrowTransactionTypeRefund,
		EscrowTransactionTypeAdjustment:
		return true
	default:
		return false
	}
}

package recurringpay

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
	case DeductionKindStandard:
		return 100
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

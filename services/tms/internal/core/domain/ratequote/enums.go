package ratequote

// Purpose says why a quote was produced, which decides how long it is kept and
// whether it is allowed to affect a shipment.
type Purpose string

const (
	// PurposeRating is a quote that priced a real shipment.
	PurposeRating = Purpose("Rating")
	// PurposeQuote priced a load that has not been booked, which is what goes
	// to the customer before there is a shipment to attach it to.
	PurposeQuote = Purpose("Quote")
	// PurposeShopping compares carriers before one is chosen.
	PurposeShopping = Purpose("Shopping")
	// PurposeSimulation is produced by replaying a proposed change against
	// history and never touches a live shipment.
	PurposeSimulation = Purpose("Simulation")
	// PurposeWhatIf answers "what would this have cost" on demand.
	PurposeWhatIf = Purpose("WhatIf")
)

func (p Purpose) String() string {
	return string(p)
}

func (p Purpose) IsValid() bool {
	switch p {
	case PurposeRating, PurposeQuote, PurposeShopping, PurposeSimulation, PurposeWhatIf:
		return true
	default:
		return false
	}
}

// AffectsShipment reports whether a quote of this purpose is allowed to set a
// shipment's charges.
func (p Purpose) AffectsShipment() bool {
	return p == PurposeRating
}

// Outcome is what the rating actually did.
//
// It carries the discriminator a shipment would otherwise need its own column
// for: whether the charge came from a contract, from a hand-picked formula
// template, from a person overriding both, or from nothing at all.
type Outcome string

const (
	OutcomeRated           = Outcome("Rated")
	OutcomeFormulaFallback = Outcome("FormulaFallback")
	OutcomeManualOverride  = Outcome("ManualOverride")
	OutcomeNoRateFound     = Outcome("NoRateFound")
	OutcomeError           = Outcome("Error")
)

func (o Outcome) String() string {
	return string(o)
}

func (o Outcome) IsValid() bool {
	switch o {
	case OutcomeRated,
		OutcomeFormulaFallback,
		OutcomeManualOverride,
		OutcomeNoRateFound,
		OutcomeError:
		return true
	default:
		return false
	}
}

// Priced reports whether the outcome produced a usable amount.
func (o Outcome) Priced() bool {
	switch o { //nolint:exhaustive // the remaining outcomes produced no amount
	case OutcomeRated, OutcomeFormulaFallback, OutcomeManualOverride:
		return true
	default:
		return false
	}
}

// NeedsAttention reports whether the outcome is one an operator should see.
func (o Outcome) NeedsAttention() bool {
	return o == OutcomeNoRateFound || o == OutcomeError
}

// Status tracks a quote's standing once it has been written.
type Status string

const (
	// StatusApplied is the quote currently governing its shipment.
	StatusApplied = Status("Applied")
	// StatusSuperseded is a quote a later rating replaced. Re-rating happens on
	// every stop edit, assignment and fuel price job, so the history matters:
	// overwriting it would destroy exactly what a dispute needs.
	StatusSuperseded = Status("Superseded")
	// StatusQuoted has no shipment behind it yet.
	StatusQuoted = Status("Quoted")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusApplied, StatusSuperseded, StatusQuoted:
		return true
	default:
		return false
	}
}

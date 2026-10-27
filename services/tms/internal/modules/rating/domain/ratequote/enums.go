package ratequote

// Purpose says why a quote was produced, which decides how long it is kept and
// whether it is allowed to affect a shipment.
type Purpose string

const (
	// Priced a real shipment.
	PurposeRating = Purpose("Rating")
	// Priced a load that has not been booked yet, as sent to the customer.
	PurposeQuote = Purpose("Quote")
	// Compared carriers before one was chosen.
	PurposeShopping = Purpose("Shopping")
	// Replayed a proposed change against history; never touches a live
	// shipment.
	PurposeSimulation = Purpose("Simulation")
	// Answered "what would this have cost" on demand.
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
	// The quote currently pricing its shipment.
	StatusApplied = Status("Applied")
	// Replaced by a later rating; kept as the history a dispute needs.
	StatusSuperseded = Status("Superseded")
	// No shipment behind it yet.
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

package rateagreement

import "slices"

// PartyType says which side of a shipment an agreement prices.
//
// One entity covers both because the structure is the same on either side —
// lanes, effective dates, versions, approval, accessorial schedules, fuel
// bindings, currency, guardrails. Splitting them would double every layer and
// let the sell and buy resolvers drift, which is precisely how margin numbers
// stop being trustworthy.
type PartyType string

const (
	PartyTypeCustomer = PartyType("Customer")
	PartyTypeCarrier  = PartyType("Carrier")
)

func (pt PartyType) String() string {
	return string(pt)
}

func (pt PartyType) IsValid() bool {
	switch pt {
	case PartyTypeCustomer, PartyTypeCarrier:
		return true
	default:
		return false
	}
}

// AgreementType describes the commercial arrangement, which drives how the
// agreement is presented and reported on rather than how it is priced.
type AgreementType string

const (
	AgreementTypeContract  = AgreementType("Contract")
	AgreementTypeTariff    = AgreementType("Tariff")
	AgreementTypeSpot      = AgreementType("Spot")
	AgreementTypeProject   = AgreementType("Project")
	AgreementTypeDedicated = AgreementType("Dedicated")
)

func (at AgreementType) String() string {
	return string(at)
}

func (at AgreementType) IsValid() bool {
	switch at {
	case AgreementTypeContract,
		AgreementTypeTariff,
		AgreementTypeSpot,
		AgreementTypeProject,
		AgreementTypeDedicated:
		return true
	default:
		return false
	}
}

// Status is an agreement's place in its review and operating lifecycle.
type Status string

const (
	StatusDraft     = Status("Draft")
	StatusInReview  = Status("InReview")
	StatusActive    = Status("Active")
	StatusSuspended = Status("Suspended")
	StatusExpired   = Status("Expired")
	StatusArchived  = Status("Archived")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft,
		StatusInReview,
		StatusActive,
		StatusSuspended,
		StatusExpired,
		StatusArchived:
		return true
	default:
		return false
	}
}

// RatesShipments reports whether an agreement in this state may price anything.
// Only Active does; everything else is either not yet approved or deliberately
// out of service.
func (s Status) RatesShipments() bool {
	return s == StatusActive
}

// allowedTransitions is the review lifecycle, in the same shape as the formula
// template state machine so the two read alike.
//
// Expired is reachable only from Active and only by the system, when the clock
// passes the agreement's end date. Archived is terminal: an archived agreement
// is kept because quotes point at it, not because it can come back.
var allowedTransitions = map[Status][]Status{
	StatusDraft:     {StatusInReview, StatusArchived},
	StatusInReview:  {StatusActive, StatusDraft, StatusArchived},
	StatusActive:    {StatusSuspended, StatusExpired, StatusArchived},
	StatusSuspended: {StatusActive, StatusArchived},
	StatusExpired:   {StatusArchived},
	StatusArchived:  {},
}

func CanTransition(from, to Status) bool {
	return from == to || slices.Contains(allowedTransitions[from], to)
}

// RuleStatus lets a single lane be taken out of service without disturbing the
// rest of the agreement.
type RuleStatus string

const (
	RuleStatusActive   = RuleStatus("Active")
	RuleStatusInactive = RuleStatus("Inactive")
)

func (rs RuleStatus) String() string {
	return string(rs)
}

func (rs RuleStatus) IsValid() bool {
	switch rs {
	case RuleStatusActive, RuleStatusInactive:
		return true
	default:
		return false
	}
}

// FreightClassSource says where a class rated rule gets its freight class.
type FreightClassSource string

const (
	// FreightClassSourceCommodity reads the class recorded on the commodity.
	FreightClassSourceCommodity = FreightClassSource("Commodity")
	// FreightClassSourceDensity derives the class from weight and cube against
	// a density scale, which is how the 2025 NMFC restructure expects most
	// freight to be classified.
	FreightClassSourceDensity = FreightClassSource("Density")
	// FreightClassSourceFixed pins every shipment on the rule to one class,
	// which is how released-value and FAK arrangements are written.
	FreightClassSourceFixed = FreightClassSource("Fixed")
)

func (fcs FreightClassSource) String() string {
	return string(fcs)
}

func (fcs FreightClassSource) IsValid() bool {
	switch fcs {
	case FreightClassSourceCommodity,
		FreightClassSourceDensity,
		FreightClassSourceFixed:
		return true
	default:
		return false
	}
}

// Direction says whether a lane prices the reverse haul as well.
type Direction string

const (
	DirectionDirectional   = Direction("Directional")
	DirectionBidirectional = Direction("Bidirectional")
)

func (d Direction) String() string {
	return string(d)
}

func (d Direction) IsValid() bool {
	switch d {
	case DirectionDirectional, DirectionBidirectional:
		return true
	default:
		return false
	}
}

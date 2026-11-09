package detentionpolicy

import (
	"errors"
)

type PolicyStatus string

const (
	PolicyStatusActive   = PolicyStatus("Active")
	PolicyStatusInactive = PolicyStatus("Inactive")
	PolicyStatusDraft    = PolicyStatus("Draft")
)

func (s PolicyStatus) String() string { return string(s) }

type ClockStartBasis string

const (
	ClockStartBasisArrival                       = ClockStartBasis("Arrival")
	ClockStartBasisAppointment                   = ClockStartBasis("Appointment")
	ClockStartBasisLaterOfArrivalOrAppointment   = ClockStartBasis("LaterOfArrivalOrAppointment")
	ClockStartBasisEarlierOfArrivalOrAppointment = ClockStartBasis("EarlierOfArrivalOrAppointment")
)

func (b ClockStartBasis) String() string { return string(b) }

type LateArrivalRule string

const (
	LateArrivalRuleNoEffect             = LateArrivalRule("NoEffect")
	LateArrivalRuleForfeit              = LateArrivalRule("Forfeit")
	LateArrivalRuleClockFromAppointment = LateArrivalRule("ClockFromAppointment")
	LateArrivalRuleReduceFreeTime       = LateArrivalRule("ReduceFreeTime")
)

func (r LateArrivalRule) String() string { return string(r) }

type RoundingMode string

const (
	RoundingModeUp      = RoundingMode("Up")
	RoundingModeDown    = RoundingMode("Down")
	RoundingModeNearest = RoundingMode("Nearest")
	RoundingModeExact   = RoundingMode("Exact")
)

func (m RoundingMode) String() string { return string(m) }

type RateSource string

const (
	RateSourceAccessorial = RateSource("Accessorial")
	RateSourceTiers       = RateSource("Tiers")
)

func (s RateSource) String() string { return string(s) }

func RateSourceFromString(v string) (RateSource, error) {
	switch v {
	case "Accessorial":
		return RateSourceAccessorial, nil
	case "Tiers":
		return RateSourceTiers, nil
	default:
		return "", errors.New("invalid detention rate source")
	}
}

type TierRateUnit string

const (
	TierRateUnitHour = TierRateUnit("Hour")
	TierRateUnitDay  = TierRateUnit("Day")
	TierRateUnitFlat = TierRateUnit("Flat")
)

func (u TierRateUnit) String() string { return string(u) }

func TierRateUnitFromString(v string) (TierRateUnit, error) {
	switch v {
	case "Hour":
		return TierRateUnitHour, nil
	case "Day":
		return TierRateUnitDay, nil
	case "Flat":
		return TierRateUnitFlat, nil
	default:
		return "", errors.New("invalid detention tier rate unit")
	}
}

type NotificationRequirement string

const (
	NotificationRequirementNone     = NotificationRequirement("None")
	NotificationRequirementAdvisory = NotificationRequirement("Advisory")
	NotificationRequirementRequired = NotificationRequirement("Required")
)

func (r NotificationRequirement) String() string { return string(r) }

type UnnotifiedBehavior string

const (
	UnnotifiedBehaviorBill     = UnnotifiedBehavior("Bill")
	UnnotifiedBehaviorFlag     = UnnotifiedBehavior("Flag")
	UnnotifiedBehaviorSuppress = UnnotifiedBehavior("Suppress")
)

func (b UnnotifiedBehavior) String() string { return string(b) }

type CapScope string

const (
	CapScopePerStop     = CapScope("PerStop")
	CapScopePerDay      = CapScope("PerDay")
	CapScopePerShipment = CapScope("PerShipment")
)

func (s CapScope) String() string { return string(s) }

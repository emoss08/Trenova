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

func PolicyStatusFromString(v string) (PolicyStatus, error) {
	switch v {
	case "Active":
		return PolicyStatusActive, nil
	case "Inactive":
		return PolicyStatusInactive, nil
	case "Draft":
		return PolicyStatusDraft, nil
	default:
		return "", errors.New("invalid detention policy status")
	}
}

type ClockStartBasis string

const (
	ClockStartBasisArrival                       = ClockStartBasis("Arrival")
	ClockStartBasisAppointment                   = ClockStartBasis("Appointment")
	ClockStartBasisLaterOfArrivalOrAppointment   = ClockStartBasis("LaterOfArrivalOrAppointment")
	ClockStartBasisEarlierOfArrivalOrAppointment = ClockStartBasis("EarlierOfArrivalOrAppointment")
)

func (b ClockStartBasis) String() string { return string(b) }

func ClockStartBasisFromString(v string) (ClockStartBasis, error) {
	switch v {
	case "Arrival":
		return ClockStartBasisArrival, nil
	case "Appointment":
		return ClockStartBasisAppointment, nil
	case "LaterOfArrivalOrAppointment":
		return ClockStartBasisLaterOfArrivalOrAppointment, nil
	case "EarlierOfArrivalOrAppointment":
		return ClockStartBasisEarlierOfArrivalOrAppointment, nil
	default:
		return "", errors.New("invalid detention clock start basis")
	}
}

type LateArrivalRule string

const (
	LateArrivalRuleNoEffect             = LateArrivalRule("NoEffect")
	LateArrivalRuleForfeit              = LateArrivalRule("Forfeit")
	LateArrivalRuleClockFromAppointment = LateArrivalRule("ClockFromAppointment")
	LateArrivalRuleReduceFreeTime       = LateArrivalRule("ReduceFreeTime")
)

func (r LateArrivalRule) String() string { return string(r) }

func LateArrivalRuleFromString(v string) (LateArrivalRule, error) {
	switch v {
	case "NoEffect":
		return LateArrivalRuleNoEffect, nil
	case "Forfeit":
		return LateArrivalRuleForfeit, nil
	case "ClockFromAppointment":
		return LateArrivalRuleClockFromAppointment, nil
	case "ReduceFreeTime":
		return LateArrivalRuleReduceFreeTime, nil
	default:
		return "", errors.New("invalid detention late arrival rule")
	}
}

type RoundingMode string

const (
	RoundingModeUp      = RoundingMode("Up")
	RoundingModeDown    = RoundingMode("Down")
	RoundingModeNearest = RoundingMode("Nearest")
	RoundingModeExact   = RoundingMode("Exact")
)

func (m RoundingMode) String() string { return string(m) }

func RoundingModeFromString(v string) (RoundingMode, error) {
	switch v {
	case "Up":
		return RoundingModeUp, nil
	case "Down":
		return RoundingModeDown, nil
	case "Nearest":
		return RoundingModeNearest, nil
	case "Exact":
		return RoundingModeExact, nil
	default:
		return "", errors.New("invalid detention rounding mode")
	}
}

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

func NotificationRequirementFromString(v string) (NotificationRequirement, error) {
	switch v {
	case "None":
		return NotificationRequirementNone, nil
	case "Advisory":
		return NotificationRequirementAdvisory, nil
	case "Required":
		return NotificationRequirementRequired, nil
	default:
		return "", errors.New("invalid detention notification requirement")
	}
}

type UnnotifiedBehavior string

const (
	UnnotifiedBehaviorBill     = UnnotifiedBehavior("Bill")
	UnnotifiedBehaviorFlag     = UnnotifiedBehavior("Flag")
	UnnotifiedBehaviorSuppress = UnnotifiedBehavior("Suppress")
)

func (b UnnotifiedBehavior) String() string { return string(b) }

func UnnotifiedBehaviorFromString(v string) (UnnotifiedBehavior, error) {
	switch v {
	case "Bill":
		return UnnotifiedBehaviorBill, nil
	case "Flag":
		return UnnotifiedBehaviorFlag, nil
	case "Suppress":
		return UnnotifiedBehaviorSuppress, nil
	default:
		return "", errors.New("invalid detention unnotified behavior")
	}
}

type CapScope string

const (
	CapScopePerStop     = CapScope("PerStop")
	CapScopePerDay      = CapScope("PerDay")
	CapScopePerShipment = CapScope("PerShipment")
)

func (s CapScope) String() string { return string(s) }

func CapScopeFromString(v string) (CapScope, error) {
	switch v {
	case "PerStop":
		return CapScopePerStop, nil
	case "PerDay":
		return CapScopePerDay, nil
	case "PerShipment":
		return CapScopePerShipment, nil
	default:
		return "", errors.New("invalid detention cap scope")
	}
}

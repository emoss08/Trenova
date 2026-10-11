package jurisdictionrule

import "errors"

var (
	ErrInvalidEscortRole   = errors.New("invalid escort role")
	ErrInvalidRestriction  = errors.New("invalid travel restriction")
	ErrInvalidTriggerKind  = errors.New("invalid trigger kind")
	ErrInvalidRuleStatus   = errors.New("invalid jurisdiction rule status")
	ErrInvalidVerification = errors.New("invalid verification state")
)

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
	StatusDraft    = Status("Draft")
)

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusDraft:
		return true
	default:
		return false
	}
}

// VerificationState records how much trust a row has earned. Seeded reference
// data ships as Unverified on purpose: it is a research baseline, not legal
// authority, and the UI says so until someone at the carrier confirms it
// against the issuing authority.
type VerificationState string

const (
	VerificationUnverified = VerificationState("Unverified")
	VerificationVerified   = VerificationState("Verified")
	VerificationDisputed   = VerificationState("Disputed")
)

func (v VerificationState) String() string { return string(v) }

func (v VerificationState) IsValid() bool {
	switch v {
	case VerificationUnverified, VerificationVerified, VerificationDisputed:
		return true
	default:
		return false
	}
}

func (v VerificationState) IsTrusted() bool {
	return v == VerificationVerified
}

type EscortRole string

const (
	EscortRoleFront  = EscortRole("Front")
	EscortRoleRear   = EscortRole("Rear")
	EscortRolePolice = EscortRole("Police")
	EscortRoleSurvey = EscortRole("RouteSurvey")
)

func (e EscortRole) String() string { return string(e) }

func (e EscortRole) IsValid() bool {
	switch e {
	case EscortRoleFront, EscortRoleRear, EscortRolePolice, EscortRoleSurvey:
		return true
	default:
		return false
	}
}

func (e EscortRole) Label() string {
	switch e {
	case EscortRoleFront:
		return "Front escort"
	case EscortRoleRear:
		return "Rear escort"
	case EscortRolePolice:
		return "Police escort"
	case EscortRoleSurvey:
		return "Route survey"
	default:
		return string(e)
	}
}

// TriggerKind names the dimension that pushed a load past a threshold. It is
// carried on every derived requirement so the user is told which number caused
// the permit, not merely that one is needed.
type TriggerKind string

const (
	TriggerWidth     = TriggerKind("Width")
	TriggerHeight    = TriggerKind("Height")
	TriggerLength    = TriggerKind("Length")
	TriggerWeight    = TriggerKind("Weight")
	TriggerSuperload = TriggerKind("Superload")
)

func (t TriggerKind) String() string { return string(t) }

func (t TriggerKind) IsValid() bool {
	switch t {
	case TriggerWidth, TriggerHeight, TriggerLength, TriggerWeight, TriggerSuperload:
		return true
	default:
		return false
	}
}

func (t TriggerKind) Unit() string {
	if t == TriggerWeight {
		return "lbs"
	}
	return "ft"
}

type RestrictionKind string

const (
	RestrictionDaylightOnly = RestrictionKind("DaylightOnly")
	RestrictionRushHour     = RestrictionKind("RushHour")
	RestrictionWeekend      = RestrictionKind("Weekend")
	RestrictionHoliday      = RestrictionKind("Holiday")
	RestrictionNightOnly    = RestrictionKind("NightOnly")
)

func (r RestrictionKind) String() string { return string(r) }

func (r RestrictionKind) IsValid() bool {
	switch r {
	case RestrictionDaylightOnly,
		RestrictionRushHour,
		RestrictionWeekend,
		RestrictionHoliday,
		RestrictionNightOnly:
		return true
	default:
		return false
	}
}

func (r RestrictionKind) Label() string {
	switch r {
	case RestrictionDaylightOnly:
		return "Daylight movement only"
	case RestrictionRushHour:
		return "Rush hour restriction"
	case RestrictionWeekend:
		return "Weekend restriction"
	case RestrictionHoliday:
		return "Holiday restriction"
	case RestrictionNightOnly:
		return "Night movement only"
	default:
		return string(r)
	}
}

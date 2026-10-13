package stopeta

type Verdict string

const (
	VerdictOnTime  = Verdict("OnTime")
	VerdictAtRisk  = Verdict("AtRisk")
	VerdictLate    = Verdict("Late")
	VerdictUnknown = Verdict("Unknown")
)

func (v Verdict) String() string { return string(v) }

func (v Verdict) IsValid() bool {
	switch v {
	case VerdictOnTime, VerdictAtRisk, VerdictLate, VerdictUnknown:
		return true
	default:
		return false
	}
}

func (v Verdict) Rank() int {
	switch v {
	case VerdictLate:
		return 3
	case VerdictAtRisk:
		return 2
	case VerdictOnTime:
		return 1
	case VerdictUnknown:
		return 0
	default:
		return 0
	}
}

func (v Verdict) NeedsAttention() bool {
	return v == VerdictLate || v == VerdictAtRisk
}

type Basis string

const (
	BasisPosition        = Basis("Position")
	BasisCheckCall       = Basis("CheckCall")
	BasisCarrierReported = Basis("CarrierReported")
	BasisAtStop          = Basis("AtStop")
	BasisLastDeparture   = Basis("LastDeparture")
	BasisNone            = Basis("None")
)

func (b Basis) String() string { return string(b) }

func (b Basis) IsValid() bool {
	switch b {
	case BasisPosition, BasisCheckCall, BasisCarrierReported, BasisAtStop, BasisLastDeparture,
		BasisNone:
		return true
	default:
		return false
	}
}

type DriveMethod string

const (
	DriveMethodRouted   = DriveMethod("Routed")
	DriveMethodPlanning = DriveMethod("Planning")
	DriveMethodReported = DriveMethod("Reported")
	DriveMethodNone     = DriveMethod("None")
)

func (m DriveMethod) String() string { return string(m) }

func (m DriveMethod) IsValid() bool {
	switch m {
	case DriveMethodRouted, DriveMethodPlanning, DriveMethodReported, DriveMethodNone:
		return true
	default:
		return false
	}
}

const (
	DefaultRiskThresholdMinutes = 60
	MinRiskThresholdMinutes     = 0
	MaxRiskThresholdMinutes     = 1440
	DefaultDwellMinutes         = 60
	MinDwellMinutes             = 0
	MaxDwellMinutes             = 720
	MaxReasonLength             = 500
)

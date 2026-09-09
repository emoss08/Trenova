package ifta

import "github.com/emoss08/trenova/pkg/domaintypes"

const MinReasonLength = 10

type JurisdictionStatus string

const (
	JurisdictionStatusActive   = JurisdictionStatus("Active")
	JurisdictionStatusInactive = JurisdictionStatus("Inactive")
)

func (s JurisdictionStatus) String() string { return string(s) }

func (s JurisdictionStatus) IsValid() bool {
	return s == JurisdictionStatusActive || s == JurisdictionStatusInactive
}

func (s JurisdictionStatus) Label() string {
	switch s {
	case JurisdictionStatusActive:
		return "Active"
	case JurisdictionStatusInactive:
		return "Inactive"
	default:
		return string(s)
	}
}

type MileageSource string

const (
	MileageSourceManual           = MileageSource("Manual")
	MileageSourceRouteCalculation = MileageSource("RouteCalculation")
	MileageSourceTelematics       = MileageSource("Telematics")
)

func (s MileageSource) String() string { return string(s) }

func (s MileageSource) IsValid() bool {
	switch s {
	case MileageSourceManual, MileageSourceRouteCalculation, MileageSourceTelematics:
		return true
	default:
		return false
	}
}

func (s MileageSource) Label() string {
	switch s {
	case MileageSourceManual:
		return "Manual entry"
	case MileageSourceRouteCalculation:
		return "Route calculation"
	case MileageSourceTelematics:
		return "Telematics"
	default:
		return string(s)
	}
}

type ReturnStatus string

const (
	ReturnStatusDraft     = ReturnStatus("Draft")
	ReturnStatusFinalized = ReturnStatus("Finalized")
	ReturnStatusFiled     = ReturnStatus("Filed")
)

func (s ReturnStatus) String() string { return string(s) }

func (s ReturnStatus) IsValid() bool {
	switch s {
	case ReturnStatusDraft, ReturnStatusFinalized, ReturnStatusFiled:
		return true
	default:
		return false
	}
}

func (s ReturnStatus) Label() string {
	switch s {
	case ReturnStatusDraft:
		return "Draft"
	case ReturnStatusFinalized:
		return "Finalized"
	case ReturnStatusFiled:
		return "Filed"
	default:
		return string(s)
	}
}

func (s ReturnStatus) CanTransitionTo(next ReturnStatus) bool {
	switch s {
	case ReturnStatusDraft:
		return next == ReturnStatusFinalized
	case ReturnStatusFinalized:
		return next == ReturnStatusDraft || next == ReturnStatusFiled
	default:
		return false
	}
}

func (s ReturnStatus) IsLocked() bool {
	return s == ReturnStatusFinalized || s == ReturnStatusFiled
}

type ProblemCode string

const (
	ProblemMissingRate          = ProblemCode("MissingRate")
	ProblemNoFuelForType        = ProblemCode("NoFuelForType")
	ProblemUnattributedMiles    = ProblemCode("UnattributedMiles")
	ProblemNoTractorMiles       = ProblemCode("NoTractorMiles")
	ProblemMileageMismatch      = ProblemCode("MileageMismatch")
	ProblemNonMemberActivity    = ProblemCode("NonMemberActivity")
	ProblemNonQualifiedActivity = ProblemCode("NonQualifiedActivity")
)

func (c ProblemCode) String() string { return string(c) }

func (c ProblemCode) IsValid() bool {
	switch c {
	case ProblemMissingRate, ProblemNoFuelForType, ProblemUnattributedMiles,
		ProblemNoTractorMiles, ProblemMileageMismatch, ProblemNonMemberActivity,
		ProblemNonQualifiedActivity:
		return true
	default:
		return false
	}
}

func (c ProblemCode) Label() string {
	switch c {
	case ProblemMissingRate:
		return "Missing tax rate"
	case ProblemNoFuelForType:
		return "No fuel purchased for fuel type"
	case ProblemUnattributedMiles:
		return "Miles not attributed to a jurisdiction"
	case ProblemNoTractorMiles:
		return "Miles without a tractor"
	case ProblemMileageMismatch:
		return "Jurisdiction miles do not match move distance"
	case ProblemNonMemberActivity:
		return "Activity in a non-member jurisdiction"
	case ProblemNonQualifiedActivity:
		return "Activity on non-qualified tractors"
	default:
		return string(c)
	}
}

func (c ProblemCode) Blocks() bool { return c == ProblemMissingRate }

type Problem struct {
	Code             ProblemCode              `json:"code"`
	Message          string                   `json:"message"`
	JurisdictionCode string                   `json:"jurisdictionCode,omitempty"`
	FuelType         domaintypes.IFTAFuelType `json:"fuelType,omitempty"`
	Amount           string                   `json:"amount,omitempty"`
}

func (p Problem) Blocks() bool { return p.Code.Blocks() }

type FleetMPG struct {
	FuelType     domaintypes.IFTAFuelType `json:"fuelType"`
	MPG          string                   `json:"mpg"`
	TotalMiles   string                   `json:"totalMiles"`
	TotalGallons string                   `json:"totalGallons"`
}

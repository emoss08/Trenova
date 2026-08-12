package permit

import "errors"

var (
	ErrInvalidPermitStatus      = errors.New("invalid permit status")
	ErrInvalidRequirementStatus = errors.New("invalid requirement status")
)

type Status string

const (
	StatusPending Status = "Pending"
	StatusActive  Status = "Active"
	StatusExpired Status = "Expired"
	StatusVoid    Status = "Void"
)

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusActive, StatusExpired, StatusVoid:
		return true
	default:
		return false
	}
}

// Satisfies reports whether a permit in this state can discharge a requirement.
// Pending deliberately does not: an application in flight is not authorization
// to move, and treating it as one is exactly the mistake that puts a truck on
// the road unpermitted.
func (s Status) Satisfies() bool {
	return s == StatusActive
}

type RequirementStatus string

const (
	RequirementOpen       RequirementStatus = "Open"
	RequirementSatisfied  RequirementStatus = "Satisfied"
	RequirementSuperseded RequirementStatus = "Superseded"
	RequirementWaived     RequirementStatus = "Waived"
)

func (s RequirementStatus) String() string { return string(s) }

func (s RequirementStatus) IsValid() bool {
	switch s {
	case RequirementOpen, RequirementSatisfied, RequirementSuperseded, RequirementWaived:
		return true
	default:
		return false
	}
}

func (s RequirementStatus) BlocksDispatch() bool {
	return s == RequirementOpen
}

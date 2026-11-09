package worker

import "errors"

var (
	ErrInvalidPTOStatus = errors.New("invalid PTO status")
	ErrInvalidPTOType   = errors.New("invalid PTO type")
)

type WorkerType string

const (
	WorkerTypeEmployee   = WorkerType("Employee")
	WorkerTypeContractor = WorkerType("Contractor")
)

func (w WorkerType) String() string {
	return string(w)
}

func (w WorkerType) IsValid() bool {
	switch w {
	case WorkerTypeEmployee, WorkerTypeContractor:
		return true
	default:
		return false
	}
}

type EndorsementType string

const (
	EndorsementTypeNone         = EndorsementType("O")
	EndorsementTypeTanker       = EndorsementType("N")
	EndorsementTypeHazmat       = EndorsementType("H")
	EndorsementTypeTankerHazmat = EndorsementType("X")
	EndorsementTypePassenger    = EndorsementType("P")
	EndorsementTypeDoubleTriple = EndorsementType("T")
)

func (e EndorsementType) String() string {
	return string(e)
}

func (e EndorsementType) IsValid() bool {
	switch e {
	case EndorsementTypeNone, EndorsementTypeTanker, EndorsementTypeHazmat,
		EndorsementTypeTankerHazmat, EndorsementTypePassenger, EndorsementTypeDoubleTriple:
		return true
	default:
		return false
	}
}

func (e EndorsementType) RequiresHazmatExpiry() bool {
	return e == EndorsementTypeHazmat || e == EndorsementTypeTankerHazmat
}

type ComplianceStatus string

const (
	ComplianceStatusCompliant    = ComplianceStatus("Compliant")
	ComplianceStatusNonCompliant = ComplianceStatus("NonCompliant")
	ComplianceStatusPending      = ComplianceStatus("Pending")
)

func (c ComplianceStatus) String() string {
	return string(c)
}

func (c ComplianceStatus) IsValid() bool {
	switch c {
	case ComplianceStatusCompliant, ComplianceStatusNonCompliant, ComplianceStatusPending:
		return true
	default:
		return false
	}
}

type PTOStatus string

const (
	PTOStatusRequested = PTOStatus("Requested")
	PTOStatusApproved  = PTOStatus("Approved")
	PTOStatusRejected  = PTOStatus("Rejected")
	PTOStatusCancelled = PTOStatus("Cancelled")
)

func (p PTOStatus) String() string {
	return string(p)
}

func (p PTOStatus) IsValid() bool {
	switch p {
	case PTOStatusRequested, PTOStatusApproved, PTOStatusRejected, PTOStatusCancelled:
		return true
	default:
		return false
	}
}

func (p PTOStatus) IsTerminal() bool {
	return p == PTOStatusRejected || p == PTOStatusCancelled
}

func (p PTOStatus) CanTransitionTo(next PTOStatus) bool {
	switch p {
	case PTOStatusRequested:
		return next == PTOStatusApproved || next == PTOStatusRejected || next == PTOStatusCancelled
	case PTOStatusApproved:
		return next == PTOStatusCancelled
	case PTOStatusRejected, PTOStatusCancelled:
		return false
	default:
		return false
	}
}

func PTOStatusSourcesFor(target PTOStatus) []PTOStatus {
	switch target {
	case PTOStatusApproved, PTOStatusRejected:
		return []PTOStatus{PTOStatusRequested}
	case PTOStatusCancelled:
		return []PTOStatus{PTOStatusRequested, PTOStatusApproved}
	case PTOStatusRequested:
		return nil
	default:
		return nil
	}
}

func PTOStatusFromString(s string) (PTOStatus, error) {
	switch s {
	case "Requested":
		return PTOStatusRequested, nil
	case "Approved":
		return PTOStatusApproved, nil
	case "Rejected":
		return PTOStatusRejected, nil
	case "Cancelled":
		return PTOStatusCancelled, nil
	default:
		return "", ErrInvalidPTOStatus
	}
}

type PTOType string

const (
	PTOTypePersonal    = PTOType("Personal")
	PTOTypeVacation    = PTOType("Vacation")
	PTOTypeSick        = PTOType("Sick")
	PTOTypeHoliday     = PTOType("Holiday")
	PTOTypeBereavement = PTOType("Bereavement")
	PTOTypeMaternity   = PTOType("Maternity")
	PTOTypePaternity   = PTOType("Paternity")
)

func (p PTOType) String() string {
	return string(p)
}

func PTOTypeValues() []PTOType {
	return []PTOType{
		PTOTypePersonal,
		PTOTypeVacation,
		PTOTypeSick,
		PTOTypeHoliday,
		PTOTypeBereavement,
		PTOTypeMaternity,
		PTOTypePaternity,
	}
}

func (p PTOType) IsValid() bool {
	switch p {
	case PTOTypePersonal, PTOTypeVacation, PTOTypeSick, PTOTypeHoliday,
		PTOTypeBereavement, PTOTypeMaternity, PTOTypePaternity:
		return true
	default:
		return false
	}
}

func PTOTypeFromString(s string) (PTOType, error) {
	switch s {
	case "Personal":
		return PTOTypePersonal, nil
	case "Vacation":
		return PTOTypeVacation, nil
	case "Sick":
		return PTOTypeSick, nil
	case "Holiday":
		return PTOTypeHoliday, nil
	case "Bereavement":
		return PTOTypeBereavement, nil
	case "Maternity":
		return PTOTypeMaternity, nil
	case "Paternity":
		return PTOTypePaternity, nil
	default:
		return "", ErrInvalidPTOType
	}
}

type Gender string

const (
	GenderMale   = Gender("Male")
	GenderFemale = Gender("Female")
)

func (g Gender) String() string {
	return string(g)
}

func (g Gender) IsValid() bool {
	switch g {
	case GenderMale, GenderFemale:
		return true
	default:
		return false
	}
}

type CDLClass string

const (
	CDLClassA = CDLClass("A")
	CDLClassB = CDLClass("B")
	CDLClassC = CDLClass("C")
)

func (c CDLClass) String() string {
	return string(c)
}

func (c CDLClass) IsValid() bool {
	switch c {
	case CDLClassA, CDLClassB, CDLClassC:
		return true
	default:
		return false
	}
}

type DriverType string

const (
	DriverTypeLocal    = DriverType("Local")
	DriverTypeRegional = DriverType("Regional")
	DriverTypeOTR      = DriverType("OTR")
	DriverTypeTeam     = DriverType("Team")
)

func (d DriverType) String() string {
	return string(d)
}

func (d DriverType) IsValid() bool {
	switch d {
	case DriverTypeLocal, DriverTypeRegional, DriverTypeOTR, DriverTypeTeam:
		return true
	default:
		return false
	}
}

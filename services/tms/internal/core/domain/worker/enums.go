package worker

import "errors"

var (
	ErrInvalidWorkerType       = errors.New("invalid worker type")
	ErrInvalidEndorsementType  = errors.New("invalid endorsement type")
	ErrInvalidComplianceStatus = errors.New("invalid compliance status")
	ErrInvalidPTOStatus        = errors.New("invalid PTO status")
	ErrInvalidPTOType          = errors.New("invalid PTO type")
	ErrInvalidGender           = errors.New("invalid gender")
	ErrInvalidCDLClass         = errors.New("invalid CDL class")
	ErrInvalidDriverType       = errors.New("invalid driver type")
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

func WorkerTypeFromString(s string) (WorkerType, error) {
	switch s {
	case "Employee":
		return WorkerTypeEmployee, nil
	case "Contractor":
		return WorkerTypeContractor, nil
	default:
		return "", ErrInvalidWorkerType
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

func EndorsementTypeFromString(s string) (EndorsementType, error) {
	switch s {
	case "O":
		return EndorsementTypeNone, nil
	case "N":
		return EndorsementTypeTanker, nil
	case "H":
		return EndorsementTypeHazmat, nil
	case "X":
		return EndorsementTypeTankerHazmat, nil
	case "P":
		return EndorsementTypePassenger, nil
	case "T":
		return EndorsementTypeDoubleTriple, nil
	default:
		return "", ErrInvalidEndorsementType
	}
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

func ComplianceStatusFromString(s string) (ComplianceStatus, error) {
	switch s {
	case "Compliant":
		return ComplianceStatusCompliant, nil
	case "NonCompliant":
		return ComplianceStatusNonCompliant, nil
	case "Pending":
		return ComplianceStatusPending, nil
	default:
		return "", ErrInvalidComplianceStatus
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

func GenderFromString(s string) (Gender, error) {
	switch s {
	case "Male":
		return GenderMale, nil
	case "Female":
		return GenderFemale, nil
	default:
		return "", ErrInvalidGender
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

func CDLClassFromString(s string) (CDLClass, error) {
	switch s {
	case "A":
		return CDLClassA, nil
	case "B":
		return CDLClassB, nil
	case "C":
		return CDLClassC, nil
	default:
		return "", ErrInvalidCDLClass
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

func DriverTypeFromString(s string) (DriverType, error) {
	switch s {
	case "Local":
		return DriverTypeLocal, nil
	case "Regional":
		return DriverTypeRegional, nil
	case "OTR":
		return DriverTypeOTR, nil
	case "Team":
		return DriverTypeTeam, nil
	default:
		return "", ErrInvalidDriverType
	}
}

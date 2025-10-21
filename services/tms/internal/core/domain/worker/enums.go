package worker

import "errors"

//nolint:revive // This is a custom type for the worker type
type WorkerType string

const (
	// WorkerTypeEmployee is the type of worker for an employee
	WorkerTypeEmployee = WorkerType("Employee")

	// WorkerTypeContractor is the type of worker for a contractor
	WorkerTypeContractor = WorkerType("Contractor")
)

type EndorsementType string

const (
	// EndorsementTypeNone is the type of endorsement for no endorsement
	EndorsementNone = EndorsementType("O")

	// EndorsementTanker is the type of endorsement for tanker endorsement
	EndorsementTanker = EndorsementType("N")

	// EndorsementHazmat is the type of endorsement for hazmat endorsement
	EndorsementHazmat = EndorsementType("H")

	// EndorsementTankerHazmat is the type of endorsement for tanker hazmat endorsement
	EndorsementTankerHazmat = EndorsementType("X")

	// EndorsementPassenger is the type of endorsement for passenger endorsement
	EndorsementPassenger = EndorsementType("P")

	// EndorsementDoublesTriples is the type of endorsement for doubles/triples endorsement
	EndorsementDoublesTriples = EndorsementType("T")
)

type PTOType string

const (
	// PTOTypePersonal is the type of PTO for personal leave
	PTOTypePersonal = PTOType("Personal")

	// PTOTypeVacation is the type of PTO for vacation leave
	PTOTypeVacation = PTOType("Vacation")

	// PTOTypeSick is the type of PTO for sick leave
	PTOTypeSick = PTOType("Sick")

	// PTOTypeHoliday is the type of PTO for holiday leave
	PTOTypeHoliday = PTOType("Holiday")

	// PTOTypeBereavement is the type of PTO for bereavement leave
	PTOTypeBereavement = PTOType("Bereavement")

	// PTOTypeMaternity is the type of PTO for maternity leave
	PTOTypeMaternity = PTOType("Maternity")

	// PTOTypePaternity is the type of PTO for paternity leave
	PTOTypePaternity = PTOType("Paternity")
)

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
		return "", errors.New("invalid PTO type")
	}
}

type PTOStatus string

const (
	// PTOStatusRequested lets the worker know that the PTO request has been requested
	// This is typically used when the PTO request is created by the worker
	PTOStatusRequested = PTOStatus("Requested")

	// PTOStatusApproved lets the worker know that the PTO request has been approved
	// This is typically used when the PTO request is approved by the manager
	PTOStatusApproved = PTOStatus("Approved")

	// PTOStatusRejected lets the worker know that the PTO request has been rejected
	// This is typically used when the PTO request is rejected by the manager
	PTOStatusRejected = PTOStatus("Rejected")

	// PTOStatusCancelled lets the worker know that the PTO request has been cancelled
	// This is typically used when the PTO request is cancelled by the worker themselves
	PTOStatusCancelled = PTOStatus("Cancelled")
)

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
		return "", errors.New("invalid PTO status")
	}
}

// DocumentType represents different types of required driver documents
type DocumentType string

const (
	// DocumentTypeMVR is the type of document for a MVR(Motor Vehicle Record)
	DocumentTypeMVR = DocumentType("MVR")

	// DocumentTypeMedicalCert is the type of document for a Medical Certificate
	DocumentTypeMedicalCert = DocumentType("MedicalCert")

	// DocumentTypeCDL is the type of document for a CDL
	DocumentTypeCDL = DocumentType("CDL")

	// DocumentTypeViolationCert is the type of document for a Violation Certificate
	DocumentTypeViolationCert = DocumentType("ViolationCert")

	// DocumentTypeEmploymentHistory is the type of document for an Employment History
	DocumentTypeEmploymentHistory = DocumentType("EmploymentHistory")

	// DocumentTypeDrugTest is the type of document for a Drug Test
	DocumentTypeDrugTest = DocumentType("DrugTest")

	// DocumentTypeRoadTest is the type of document for a Road Test
	DocumentTypeRoadTest = DocumentType("RoadTest")

	// DocumentTypeTrainingCert is the type of document for a Training Certificate
	DocumentTypeTrainingCert = DocumentType("TrainingCert")
)

// DocumentStatus represents the current status of a document
type DocumentStatus string

const (
	// DocumentStatusPending lets the worker know that the document is pending
	DocumentStatusPending = DocumentStatus("Pending")

	// DocumentStatusActive lets the worker know that the document is active
	DocumentStatusActive = DocumentStatus("Active")

	// DocumentStatusExpired lets the worker know that the document is expired
	DocumentStatusExpired = DocumentStatus("Expired")

	// DocumentStatusRejected lets the worker know that the document is rejected
	DocumentStatusRejected = DocumentStatus("Rejected")

	// DocumentStatusRevoked lets the worker know that the document is revoked
	DocumentStatusRevoked = DocumentStatus("Revoked")
)

// VerificationStatus represents the verification state of a document
type VerificationStatus string

const (
	// VerificationStatusPending lets the worker know that the document is pending
	VerificationStatusPending = VerificationStatus("Pending")

	// VerificationStatusVerified lets the worker know that the document is verified
	VerificationStatusVerified = VerificationStatus("Verified")

	// VerificationStatusRejected lets the worker know that the document is rejected
	VerificationStatusRejected = VerificationStatus("Rejected")

	// VerificationStatusIncomplete lets the worker know that the document is incomplete
	VerificationStatusIncomplete = VerificationStatus("Incomplete")
)

type DocumentRequirementType string

const (
	// RequirementTypeOngoing represents documents that need periodic renewal
	RequirementTypeOngoing = DocumentRequirementType("Ongoing")

	// RequirementTypeOneTime represents documents that are collected once
	RequirementTypeOneTime = DocumentRequirementType("OneTime")

	// RequirementTypeConditional represents documents that are required based on certain conditions
	RequirementTypeConditional = DocumentRequirementType("Conditional")
)

type RetentionPeriod string

const (
	// RetentionPeriodThreeYears represents the standard 3-year retention period
	RetentionPeriodThreeYears = RetentionPeriod("3Years")

	// RetentionPeriodLifeOfEmployment represents retention for employment duration plus 3 years
	RetentionPeriodLifeOfEmployment = RetentionPeriod("LifeOfEmployment")

	// RetentionPeriodCustom represents a custom retention period
	RetentionPeriodCustom = RetentionPeriod("Custom")
)

type ComplianceStatus string

const (
	// ComplianceStatusCompliant lets the worker know that the worker is compliant
	ComplianceStatusCompliant = ComplianceStatus("Compliant")

	// ComplianceStatusNonCompliant lets the worker know that the worker is non-compliant
	ComplianceStatusNonCompliant = ComplianceStatus("NonCompliant")

	// ComplianceStatusPending lets the worker know that the worker is pending
	ComplianceStatusPending = ComplianceStatus("Pending")
)

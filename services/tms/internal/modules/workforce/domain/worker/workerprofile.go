package worker

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Licensing, compliance, and qualification details for a worker
type WorkerProfile struct {
	keel.Table `keel:"worker_profiles,report=Parties"`

	ID                        pulid.ID           `json:"id"                        keel:"id,pk,autoid:wrkp_"`
	WorkerID                  pulid.ID           `json:"workerId"                  keel:"worker_id,pk"`
	BusinessUnitID            pulid.ID           `json:"businessUnitId"            keel:"business_unit_id,pk,tenant"`
	OrganizationID            pulid.ID           `json:"organizationId"            keel:"organization_id,pk,tenant"`
	LicenseStateID            pulid.ID           `json:"licenseStateId"            keel:"license_state_id,nullzero"`
	DOB                       int64              `json:"dob"                       keel:"dob,label=Date of Birth,type=epoch"`
	LicenseNumber             string             `json:"licenseNumber"             keel:"license_number"`
	CDLClass                  CDLClass           `json:"cdlClass"                  keel:"cdl_class"`
	CDLRestrictions           string             `json:"cdlRestrictions"           keel:"cdl_restrictions,nullzero"`
	Endorsement               EndorsementType    `json:"endorsement"               keel:"endorsement"`
	HazmatExpiry              *int64             `json:"hazmatExpiry"              keel:"hazmat_expiry,type=epoch"`
	LicenseExpiry             int64              `json:"licenseExpiry"             keel:"license_expiry,type=epoch"`
	MedicalCardExpiry         *int64             `json:"medicalCardExpiry"         keel:"medical_card_expiry,type=epoch"`
	MedicalExaminerName       string             `json:"medicalExaminerName"       keel:"medical_examiner_name,nullzero"`
	MedicalExaminerNPI        string             `json:"medicalExaminerNpi"        keel:"medical_examiner_npi,nullzero"`
	TWICCardNumber            string             `json:"twicCardNumber"            keel:"twic_card_number,nullzero"`
	TWICExpiry                *int64             `json:"twicExpiry"                keel:"twic_expiry,type=epoch"`
	HireDate                  int64              `json:"hireDate"                  keel:"hire_date"`
	TerminationDate           *int64             `json:"terminationDate"           keel:"termination_date"`
	PhysicalDueDate           *int64             `json:"physicalDueDate"           keel:"physical_due_date"`
	MVRDueDate                *int64             `json:"mvrDueDate"                keel:"mvr_due_date"`
	ComplianceStatus          ComplianceStatus   `json:"complianceStatus"          keel:"compliance_status"`
	TrainingHealth            TrainingHealth     `json:"trainingHealth"            keel:"training_health"`
	SafetyRating              SafetyRating       `json:"safetyRating"              keel:"safety_rating"`
	SafetyScore               int16              `json:"safetyScore"               keel:"safety_score,dimension"`
	NextCredentialExpiry      *int64             `json:"nextCredentialExpiry"      keel:"next_credential_expiry,type=epoch"`
	NextTrainingDue           *int64             `json:"nextTrainingDue"           keel:"next_training_due,type=epoch"`
	IsQualified               bool               `json:"isQualified"               keel:"is_qualified,label=Qualified"`
	DisqualificationReason    string             `json:"disqualificationReason"    keel:"disqualification_reason,nullzero"`
	LastComplianceCheck       int64              `json:"lastComplianceCheck"       keel:"last_compliance_check,type=epoch"`
	LastMVRCheck              int64              `json:"lastMvrCheck"              keel:"last_mvr_check,type=epoch"`
	LastDrugTest              int64              `json:"lastDrugTest"              keel:"last_drug_test,type=epoch"`
	DrugAlcoholStatus         DrugAlcoholStatus  `json:"drugAlcoholStatus"         keel:"drug_alcohol_status"`
	ReturnToDutyStatus        ReturnToDutyStatus `json:"returnToDutyStatus"        keel:"return_to_duty_status"`
	LastClearinghouseQueryAt  *int64             `json:"lastClearinghouseQueryAt"  keel:"last_clearinghouse_query_at"`
	NextClearinghouseQueryDue *int64             `json:"nextClearinghouseQueryDue" keel:"next_clearinghouse_query_due,type=epoch"`
	ELDExempt                 bool               `json:"eldExempt"                 keel:"eld_exempt"`
	ShortHaulExempt           bool               `json:"shortHaulExempt"           keel:"short_haul_exempt,label=Short-Haul Exempt"`
	Version                   int64              `json:"version"                   keel:"version,version"`
	CreatedAt                 int64              `json:"createdAt"                 keel:"created_at,autocreate"`
	UpdatedAt                 int64              `json:"updatedAt"                 keel:"updated_at,autoupdate"`

	LicenseState *usstate.UsState `json:"licenseState,omitempty" keel:"belongs:license_state_id,noreport"`
}

func (wp *WorkerProfile) BeforeCreate(_ context.Context) error {
	if wp.CDLClass == "" {
		wp.CDLClass = CDLClassA
	}
	if wp.Endorsement == "" {
		wp.Endorsement = EndorsementTypeNone
	}
	if wp.ComplianceStatus == "" {
		wp.ComplianceStatus = ComplianceStatusPending
	}
	if wp.TrainingHealth == "" {
		wp.TrainingHealth = TrainingHealthCurrent
	}
	if wp.SafetyRating == "" {
		wp.SafetyRating = SafetyRatingExcellent
	}
	if wp.SafetyScore == 0 {
		wp.SafetyScore = 100
	}
	if wp.DrugAlcoholStatus == "" {
		wp.DrugAlcoholStatus = DrugAlcoholUnknown
	}
	if wp.ReturnToDutyStatus == "" {
		wp.ReturnToDutyStatus = ReturnToDutyNotRequired
	}

	return nil
}

func (wp *WorkerProfile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(wp,
		validation.Field(&wp.DOB,
			validation.Required.Error("Date of birth is required"),
			validation.Min(int64(0)).Error("Date of birth must be a positive value"),
		),
		validation.Field(&wp.LicenseNumber,
			validation.Required.Error("License number is required"),
			validation.Length(1, 50).Error("License number must be between 1 and 50 characters"),
		),
		validation.Field(
			&wp.Endorsement,
			validation.Required.Error("Endorsement type is required"),
			domainvalidation.ValidEnum[EndorsementType](
				"endorsement must be one of: O, N, H, X, P, T",
			),
		),
		validation.Field(&wp.LicenseExpiry,
			validation.Required.Error("License expiry is required"),
			validation.Min(int64(1)).Error("License expiry must be a positive value"),
		),
		validation.Field(&wp.HireDate,
			validation.Required.Error("Hire date is required"),
			validation.Min(int64(1)).Error("Hire date must be a positive value"),
		),
		validation.Field(
			&wp.ComplianceStatus,
			validation.Required.Error("Compliance status is required"),
			domainvalidation.ValidEnum[ComplianceStatus](
				"compliance status must be one of: Compliant, NonCompliant, Pending",
			),
		),
	))

	if wp.Endorsement.RequiresHazmatExpiry() && (wp.HazmatExpiry == nil || *wp.HazmatExpiry <= 0) {
		multiErr.Add(
			"hazmatExpiry",
			errortypes.ErrRequired,
			"Hazmat expiry is required when endorsement is H or X",
		)
	}
}

func (wp *WorkerProfile) GetTableName() string {
	return "worker_profiles"
}

func (WorkerProfile) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_worker_profiles_license_state",
			Field:   "profile.licenseStateId",
			Message: "License state does not exist",
		},
	}
}

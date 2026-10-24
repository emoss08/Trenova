package worker

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*WorkerProfile)(nil)

// Licensing, compliance, and qualification details for a worker
type WorkerProfile struct {
	bun.BaseModel `bun:"table:worker_profiles,alias:wrkp" json:"-"`
	keel.Table    `                                                keel:"worker_profiles,report=Parties"`

	ID                        pulid.ID           `json:"id"                        bun:"id,pk,type:VARCHAR(100)"                                                                    keel:"id,pk,autoid:wrkp_"`
	WorkerID                  pulid.ID           `json:"workerId"                  bun:"worker_id,pk,type:VARCHAR(100),notnull"                                                     keel:"worker_id,pk"`
	BusinessUnitID            pulid.ID           `json:"businessUnitId"            bun:"business_unit_id,type:VARCHAR(100),notnull,pk"                                              keel:"business_unit_id,pk,tenant"`
	OrganizationID            pulid.ID           `json:"organizationId"            bun:"organization_id,type:VARCHAR(100),notnull,pk"                                               keel:"organization_id,pk,tenant"`
	LicenseStateID            pulid.ID           `json:"licenseStateId"            bun:"license_state_id,type:VARCHAR(100),nullzero"                                                keel:"license_state_id,nullzero"`
	DOB                       int64              `json:"dob"                       bun:"dob,type:BIGINT,notnull"                                                                    keel:"dob,label=Date of Birth,type=epoch"`
	LicenseNumber             string             `json:"licenseNumber"             bun:"license_number,type:VARCHAR(50),notnull"                                                    keel:"license_number"`
	CDLClass                  CDLClass           `json:"cdlClass"                  bun:"cdl_class,type:cdl_class_enum,notnull,default:'A'"                                          keel:"cdl_class"`
	CDLRestrictions           string             `json:"cdlRestrictions"           bun:"cdl_restrictions,type:VARCHAR(100),nullzero"                                                keel:"cdl_restrictions,nullzero"`
	Endorsement               EndorsementType    `json:"endorsement"               bun:"endorsement,type:endorsement_type_enum,notnull,default:'O'"                                 keel:"endorsement"`
	HazmatExpiry              *int64             `json:"hazmatExpiry"              bun:"hazmat_expiry,type:BIGINT,nullzero"                                                         keel:"hazmat_expiry,type=epoch"`
	LicenseExpiry             int64              `json:"licenseExpiry"             bun:"license_expiry,type:BIGINT,notnull"                                                         keel:"license_expiry,type=epoch"`
	MedicalCardExpiry         *int64             `json:"medicalCardExpiry"         bun:"medical_card_expiry,type:BIGINT,nullzero"                                                   keel:"medical_card_expiry,type=epoch"`
	MedicalExaminerName       string             `json:"medicalExaminerName"       bun:"medical_examiner_name,type:VARCHAR(100),nullzero"                                           keel:"medical_examiner_name,nullzero"`
	MedicalExaminerNPI        string             `json:"medicalExaminerNpi"        bun:"medical_examiner_npi,type:VARCHAR(20),nullzero"                                             keel:"medical_examiner_npi,nullzero"`
	TWICCardNumber            string             `json:"twicCardNumber"            bun:"twic_card_number,type:VARCHAR(50),nullzero"                                                 keel:"twic_card_number,nullzero"`
	TWICExpiry                *int64             `json:"twicExpiry"                bun:"twic_expiry,type:BIGINT,nullzero"                                                           keel:"twic_expiry,type=epoch"`
	HireDate                  int64              `json:"hireDate"                  bun:"hire_date,type:BIGINT,notnull"                                                              keel:"hire_date"`
	TerminationDate           *int64             `json:"terminationDate"           bun:"termination_date,type:BIGINT,nullzero"                                                      keel:"termination_date"`
	PhysicalDueDate           *int64             `json:"physicalDueDate"           bun:"physical_due_date,type:BIGINT,nullzero"                                                     keel:"physical_due_date"`
	MVRDueDate                *int64             `json:"mvrDueDate"                bun:"mvr_due_date,type:BIGINT,nullzero"                                                          keel:"mvr_due_date"`
	ComplianceStatus          ComplianceStatus   `json:"complianceStatus"          bun:"compliance_status,type:compliance_status_enum,notnull,default:'Pending'"                    keel:"compliance_status"`
	TrainingHealth            TrainingHealth     `json:"trainingHealth"            bun:"training_health,type:worker_training_health_enum,notnull,default:'Current'"                 keel:"training_health"`
	SafetyRating              SafetyRating       `json:"safetyRating"              bun:"safety_rating,type:worker_safety_rating_enum,notnull,default:'Excellent'"                   keel:"safety_rating"`
	SafetyScore               int16              `json:"safetyScore"               bun:"safety_score,type:SMALLINT,notnull,default:100"                                             keel:"safety_score,dimension"`
	NextCredentialExpiry      *int64             `json:"nextCredentialExpiry"      bun:"next_credential_expiry,type:BIGINT,nullzero"                                                keel:"next_credential_expiry,type=epoch"`
	NextTrainingDue           *int64             `json:"nextTrainingDue"           bun:"next_training_due,type:BIGINT,nullzero"                                                     keel:"next_training_due,type=epoch"`
	IsQualified               bool               `json:"isQualified"               bun:"is_qualified,type:BOOLEAN,notnull"                                                          keel:"is_qualified,label=Qualified"`
	DisqualificationReason    string             `json:"disqualificationReason"    bun:"disqualification_reason,type:VARCHAR(255),nullzero"                                         keel:"disqualification_reason,nullzero"`
	LastComplianceCheck       int64              `json:"lastComplianceCheck"       bun:"last_compliance_check,type:BIGINT,notnull"                                                  keel:"last_compliance_check,type=epoch"`
	LastMVRCheck              int64              `json:"lastMvrCheck"              bun:"last_mvr_check,type:BIGINT,notnull"                                                         keel:"last_mvr_check,type=epoch"`
	LastDrugTest              int64              `json:"lastDrugTest"              bun:"last_drug_test,type:BIGINT,notnull"                                                         keel:"last_drug_test,type=epoch"`
	DrugAlcoholStatus         DrugAlcoholStatus  `json:"drugAlcoholStatus"         bun:"drug_alcohol_status,type:worker_drug_alcohol_status_enum,notnull,default:'Unknown'"         keel:"drug_alcohol_status"`
	ReturnToDutyStatus        ReturnToDutyStatus `json:"returnToDutyStatus"        bun:"return_to_duty_status,type:worker_return_to_duty_status_enum,notnull,default:'NotRequired'" keel:"return_to_duty_status"`
	LastClearinghouseQueryAt  *int64             `json:"lastClearinghouseQueryAt"  bun:"last_clearinghouse_query_at,type:BIGINT,nullzero"                                           keel:"last_clearinghouse_query_at"`
	NextClearinghouseQueryDue *int64             `json:"nextClearinghouseQueryDue" bun:"next_clearinghouse_query_due,type:BIGINT,nullzero"                                          keel:"next_clearinghouse_query_due,type=epoch"`
	ELDExempt                 bool               `json:"eldExempt"                 bun:"eld_exempt,type:BOOLEAN,notnull"                                                            keel:"eld_exempt"`
	ShortHaulExempt           bool               `json:"shortHaulExempt"           bun:"short_haul_exempt,type:BOOLEAN,notnull"                                                     keel:"short_haul_exempt,label=Short-Haul Exempt"`
	Version                   int64              `json:"version"                   bun:"version,type:BIGINT"                                                                        keel:"version,version"`
	CreatedAt                 int64              `json:"createdAt"                 bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"                   keel:"created_at,autocreate"`
	UpdatedAt                 int64              `json:"updatedAt"                 bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"                   keel:"updated_at,autoupdate"`

	LicenseState *usstate.UsState `json:"licenseState,omitempty" bun:"rel:belongs-to,join:license_state_id=id" keel:"belongs:license_state_id,noreport"`
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

func (wp *WorkerProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if wp.ID.IsNil() {
			wp.ID = pulid.MustNew("wrkp_")
		}
		wp.CreatedAt = now
	case *bun.UpdateQuery:
		wp.UpdatedAt = now
	}

	return nil
}

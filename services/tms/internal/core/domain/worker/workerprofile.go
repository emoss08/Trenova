package worker

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*WorkerProfile)(nil)

type WorkerProfile struct {
	bun.BaseModel `bun:"table:worker_profiles,alias:wrkp" json:"-"`

	ID                     pulid.ID         `json:"id"                     bun:"id,pk,type:VARCHAR(100)"`
	WorkerID               pulid.ID         `json:"workerId"               bun:"worker_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID         `json:"businessUnitId"         bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID         pulid.ID         `json:"organizationId"         bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	LicenseStateID         pulid.ID         `json:"licenseStateId"         bun:"license_state_id,type:VARCHAR(100),nullzero"`
	DOB                    int64            `json:"dob"                    bun:"dob,type:BIGINT,notnull"`
	LicenseNumber          string           `json:"licenseNumber"          bun:"license_number,type:VARCHAR(50),notnull"`
	CDLClass               CDLClass         `json:"cdlClass"               bun:"cdl_class,type:cdl_class_enum,notnull,default:'A'"`
	CDLRestrictions        string           `json:"cdlRestrictions"        bun:"cdl_restrictions,type:VARCHAR(100),nullzero"`
	Endorsement            EndorsementType  `json:"endorsement"            bun:"endorsement,type:endorsement_type_enum,notnull,default:'O'"`
	HazmatExpiry           *int64           `json:"hazmatExpiry"           bun:"hazmat_expiry,type:BIGINT,nullzero"`
	LicenseExpiry          int64            `json:"licenseExpiry"          bun:"license_expiry,type:BIGINT,notnull"`
	MedicalCardExpiry      *int64           `json:"medicalCardExpiry"      bun:"medical_card_expiry,type:BIGINT,nullzero"`
	MedicalExaminerName    string           `json:"medicalExaminerName"    bun:"medical_examiner_name,type:VARCHAR(100),nullzero"`
	MedicalExaminerNPI     string           `json:"medicalExaminerNpi"     bun:"medical_examiner_npi,type:VARCHAR(20),nullzero"`
	TWICCardNumber         string           `json:"twicCardNumber"         bun:"twic_card_number,type:VARCHAR(50),nullzero"`
	TWICExpiry             *int64           `json:"twicExpiry"             bun:"twic_expiry,type:BIGINT,nullzero"`
	HireDate               int64            `json:"hireDate"               bun:"hire_date,type:BIGINT,notnull"`
	TerminationDate        *int64           `json:"terminationDate"        bun:"termination_date,type:BIGINT,nullzero"`
	PhysicalDueDate        *int64           `json:"physicalDueDate"        bun:"physical_due_date,type:BIGINT,nullzero"`
	MVRDueDate             *int64           `json:"mvrDueDate"             bun:"mvr_due_date,type:BIGINT,nullzero"`
	ComplianceStatus       ComplianceStatus `json:"complianceStatus"       bun:"compliance_status,type:compliance_status_enum,notnull,default:'Pending'"`
	IsQualified            bool             `json:"isQualified"            bun:"is_qualified,type:BOOLEAN,notnull,default:false"`
	DisqualificationReason string           `json:"disqualificationReason" bun:"disqualification_reason,type:VARCHAR(255),nullzero"`
	LastComplianceCheck    int64            `json:"lastComplianceCheck"    bun:"last_compliance_check,type:BIGINT,notnull,default:0"`
	LastMVRCheck           int64            `json:"lastMvrCheck"           bun:"last_mvr_check,type:BIGINT,notnull,default:0"`
	LastDrugTest           int64            `json:"lastDrugTest"           bun:"last_drug_test,type:BIGINT,notnull,default:0"`
	ELDExempt              bool             `json:"eldExempt"              bun:"eld_exempt,type:BOOLEAN,notnull,default:false"`
	ShortHaulExempt        bool             `json:"shortHaulExempt"        bun:"short_haul_exempt,type:BOOLEAN,notnull,default:false"`
	Version                int64            `json:"version"                bun:"version,type:BIGINT"`
	CreatedAt              int64            `json:"createdAt"              bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64            `json:"updatedAt"              bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	LicenseState *usstate.UsState `json:"licenseState,omitempty" bun:"rel:belongs-to,join:license_state_id=id"`
}

func (wp *WorkerProfile) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(wp,
		validation.Field(&wp.DOB,
			validation.Required.Error("Date of birth is required"),
			validation.Min(int64(0)).Error("Date of birth must be a positive value"),
		),
		validation.Field(&wp.LicenseNumber,
			validation.Required.Error("License number is required"),
			validation.Length(1, 50).Error("License number must be between 1 and 50 characters"),
		),
		validation.Field(&wp.Endorsement,
			validation.Required.Error("Endorsement type is required"),
			validation.By(func(value any) error {
				e, ok := value.(EndorsementType)
				if !ok {
					return errors.New("invalid endorsement type")
				}
				if !e.IsValid() {
					return errors.New("endorsement must be one of: O, N, H, X, P, T")
				}
				return nil
			}),
		),
		validation.Field(&wp.LicenseExpiry,
			validation.Required.Error("License expiry is required"),
			validation.Min(int64(1)).Error("License expiry must be a positive value"),
		),
		validation.Field(&wp.HireDate,
			validation.Required.Error("Hire date is required"),
			validation.Min(int64(1)).Error("Hire date must be a positive value"),
		),
		validation.Field(&wp.ComplianceStatus,
			validation.Required.Error("Compliance status is required"),
			validation.By(func(value any) error {
				cs, ok := value.(ComplianceStatus)
				if !ok {
					return errors.New("invalid compliance status type")
				}
				if !cs.IsValid() {
					return errors.New(
						"compliance status must be one of: Compliant, NonCompliant, Pending",
					)
				}
				return nil
			}),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

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

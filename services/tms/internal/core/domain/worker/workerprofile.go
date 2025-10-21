package worker

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*WorkerProfile)(nil)

//nolint:revive // struct should keep this name
type WorkerProfile struct {
	bun.BaseModel `bun:"table:worker_profiles,alias:wp" json:"-"`

	ID                     pulid.ID         `bun:"id,type:VARCHAR(100),pk,notnull"                                          json:"id"`
	WorkerID               pulid.ID         `bun:"worker_id,type:VARCHAR(100),notnull"                                      json:"workerId"`
	BusinessUnitID         pulid.ID         `bun:"business_unit_id,type:VARCHAR(100),notnull"                               json:"businessUnitId"`
	OrganizationID         pulid.ID         `bun:"organization_id,type:VARCHAR(100),pk,notnull"                             json:"organizationId"`
	LicenseStateID         pulid.ID         `bun:"license_state_id,type:VARCHAR(100)"                                       json:"licenseStateId"`
	DOB                    int64            `bun:"dob,type:BIGINT,notnull"                                                  json:"dob"`
	LicenseNumber          string           `bun:"license_number,type:VARCHAR(50),notnull"                                  json:"licenseNumber"`
	Endorsement            EndorsementType  `bun:"endorsement,type:endorsement_type_enum,notnull,default:'O'"               json:"endorsement"`
	HazmatExpiry           int64            `bun:"hazmat_expiry,type:BIGINT"                                                json:"hazmatExpiry"`
	LicenseExpiry          int64            `bun:"license_expiry,type:BIGINT,notnull"                                       json:"licenseExpiry"`
	HireDate               int64            `bun:"hire_date,type:BIGINT,notnull"                                            json:"hireDate"`
	TerminationDate        *int64           `bun:"termination_date,type:BIGINT,nullzero"                                    json:"terminationDate"`
	PhysicalDueDate        *int64           `bun:"physical_due_date,type:BIGINT,nullzero"                                   json:"physicalDueDate"`
	MVRDueDate             *int64           `bun:"mvr_due_date,type:BIGINT,nullzero"                                        json:"mvrDueDate"`
	ComplianceStatus       ComplianceStatus `bun:"compliance_status,type:compliance_status_enum,notnull,default:'Pending'"  json:"complianceStatus"`
	IsQualified            bool             `bun:"is_qualified,type:BOOLEAN,notnull,default:true"                           json:"isQualified"`
	DisqualificationReason string           `bun:"disqualification_reason,type:TEXT"                                        json:"disqualificationReason,omitempty"`
	LastComplianceCheck    int64            `bun:"last_compliance_check,type:BIGINT,notnull"                                json:"lastComplianceCheck"`
	LastMVRCheck           int64            `bun:"last_mvr_check,type:BIGINT,notnull"                                       json:"lastMvrCheck"`
	LastDrugTest           int64            `bun:"last_drug_test,type:BIGINT,notnull"                                       json:"lastDrugTest"`
	Version                int64            `bun:"version,type:BIGINT"                                                      json:"version"`
	CreatedAt              int64            `bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt              int64            `bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	LicenseState *usstate.UsState     `json:"licenseState,omitempty" bun:"rel:belongs-to,join:license_state_id=id"`
}

func (p *WorkerProfile) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(p,
		validation.Field(&p.Endorsement,
			validation.Required.Error("Endorsement is required"),
			validation.In(
				EndorsementNone,
				EndorsementTanker,
				EndorsementHazmat,
				EndorsementTankerHazmat,
				EndorsementPassenger,
				EndorsementDoublesTriples,
			).Error("Endorsement must be a valid type"),
		),
		validation.Field(&p.HireDate,
			validation.Required.Error("Hire date is required"),
		),
		validation.Field(&p.HazmatExpiry,
			validation.Required.When(
				p.Endorsement == EndorsementHazmat || p.Endorsement == EndorsementTankerHazmat,
			).Error("Hazmat expiry is required for hazmat and tanker hazmat endorsements"),
		),
		validation.Field(&p.LicenseExpiry,
			validation.Required.Error("License expiry is required"),
		),
		validation.Field(&p.ComplianceStatus,
			validation.Required.Error("Compliance status is required"),
			validation.In(
				ComplianceStatusCompliant,
				ComplianceStatusNonCompliant,
				ComplianceStatusPending,
			).Error("Invalid compliance status"),
		),
		validation.Field(&p.LastDrugTest,
			validation.Required.Error("Last drug test is required"),
		),
		validation.Field(&p.LastMVRCheck,
			validation.Required.Error("Last MVR check is required"),
		),
		validation.Field(&p.TerminationDate,
			validation.When(p.TerminationDate != nil,
				validation.Min(p.HireDate).Error("Termination date must be after hire date"),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (p *WorkerProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("wp_")
		}

		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

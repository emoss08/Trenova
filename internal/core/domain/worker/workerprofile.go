package worker

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*WorkerProfile)(nil)

//nolint:revive // struct should keep this name
type WorkerProfile struct {
	bun.BaseModel `bun:"table:worker_profiles,alias:wp" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	WorkerID       pulid.ID `bun:"worker_id,type:VARCHAR(100),notnull" json:"workerId"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`
	LicenseStateID pulid.ID `bun:"license_state_id,type:VARCHAR(100)" json:"licenseStateId"`

	// Core Fields
	DOB                    int64            `json:"dob" bun:"dob,type:BIGINT,notnull"`
	LicenseNumber          string           `json:"licenseNumber" bun:"license_number,type:VARCHAR(50),notnull"`
	Endorsement            EndorsementType  `json:"endorsement" bun:"endorsement,type:endorsement_type_enum,notnull,default:'O'"`
	HazmatExpiry           int64            `json:"hazmatExpiry" bun:"hazmat_expiry,type:BIGINT"`
	LicenseExpiry          int64            `json:"licenseExpiry" bun:"license_expiry,type:BIGINT,notnull"`
	HireDate               int64            `json:"hireDate" bun:"hire_date,type:BIGINT,notnull"`
	TerminationDate        *int64           `json:"terminationDate" bun:"termination_date,type:BIGINT,nullzero"`
	PhysicalDueDate        *int64           `json:"physicalDueDate" bun:"physical_due_date,type:BIGINT,nullzero"`
	MVRDueDate             *int64           `json:"mvrDueDate" bun:"mvr_due_date,type:BIGINT,nullzero"`
	ComplianceStatus       ComplianceStatus `json:"complianceStatus" bun:"compliance_status,type:compliance_status_enum,notnull,default:'Pending'"`
	IsQualified            bool             `json:"isQualified" bun:"is_qualified,type:BOOLEAN,notnull,default:true"`
	DisqualificationReason string           `json:"disqualificationReason,omitempty" bun:"disqualification_reason,type:TEXT"`
	LastComplianceCheck    int64            `json:"lastComplianceCheck" bun:"last_compliance_check,type:BIGINT,notnull"`
	LastMVRCheck           int64            `json:"lastMvrCheck" bun:"last_mvr_check,type:BIGINT,notnull"`
	LastDrugTest           int64            `json:"lastDrugTest" bun:"last_drug_test,type:BIGINT,notnull"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	LicenseState *usstate.UsState           `json:"licenseState,omitempty" bun:"rel:belongs-to,join:license_state_id=id"`
}

func (p *WorkerProfile) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, p,
		// Ensure the endorsement is valid
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

		// Ensure the hire date is valid and not in the future
		validation.Field(&p.HireDate,
			validation.Required.Error("Hire date is required"),
		),

		// If the endorsement is H (Hazmat), or X (Tanker Hazmat), then the hazmat expiry is required
		validation.Field(&p.HazmatExpiry,
			validation.Required.When(
				p.Endorsement == EndorsementHazmat || p.Endorsement == EndorsementTankerHazmat,
			).Error("Hazmat expiry is required for hazmat and tanker hazmat endorsements"),
		),

		// Ensure the license expiry is required
		validation.Field(&p.LicenseExpiry,
			validation.Required.Error("License expiry is required"),
		),

		// Compliance status is required and must be a valid compliance statuss
		validation.Field(&p.ComplianceStatus,
			validation.Required.Error("Compliance status is required"),
			validation.In(
				ComplianceStatusCompliant,
				ComplianceStatusNonCompliant,
				ComplianceStatusPending,
			).Error("Invalid compliance status"),
		),

		// Last drug test is required
		validation.Field(&p.LastDrugTest,
			validation.Required.Error("Last drug test is required"),
		),

		validation.Field(&p.LastMVRCheck,
			validation.Required.Error("Last MVR check is required"),
		),

		// Termination date must be after hire date
		validation.Field(&p.TerminationDate,
			validation.When(p.TerminationDate != nil,
				validation.Min(p.HireDate).Error("Termination date must be after hire date"),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (p *WorkerProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := time.Now().Unix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID == "" {
			p.ID = pulid.MustNew("wp_")
		}

		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

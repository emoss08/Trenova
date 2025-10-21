package tenant

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*DispatchControl)(nil)
	_ domain.Validatable        = (*DispatchControl)(nil)
	_ framework.TenantedEntity  = (*DispatchControl)(nil)
)

type DispatchControl struct {
	bun.BaseModel `bun:"table:dispatch_controls,alias:dc" json:"-"`

	ID                                   pulid.ID                   `json:"id"                                   bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID                       pulid.ID                   `json:"businessUnitId"                       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID                       pulid.ID                   `json:"organizationId"                       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EnableAutoAssignment                 bool                       `json:"enableAutoAssignment"                 bun:"enable_auto_assignment,type:BOOLEAN,notnull,default:true"`
	EnforceHOSCompliance                 bool                       `json:"enforceHosCompliance"                 bun:"enforce_hos_compliance,type:BOOLEAN,notnull,default:true"`
	EnforceWorkerAssign                  bool                       `json:"enforceWorkerAssign"                  bun:"enforce_worker_assign,type:BOOLEAN,notnull,default:true"`
	EnforceTrailerContinuity             bool                       `json:"enforceTrailerContinuity"             bun:"enforce_trailer_continuity,type:BOOLEAN,notnull,default:true"`
	EnforceWorkerPTARestrictions         bool                       `json:"enforceWorkerPTARestrictions"         bun:"enforce_worker_pta_restrictions,type:BOOLEAN,notnull,default:true"`
	EnforceWorkerTractorFleetContinuity  bool                       `json:"enforceWorkerTractorFleetContinuity"  bun:"enforce_worker_tractor_fleet_continuity,type:BOOLEAN,notnull,default:true"`
	EnforceDriverQualificationCompliance bool                       `json:"enforceDriverQualificationCompliance" bun:"enforce_driver_qualification_compliance,type:BOOLEAN,notnull,default:true"`
	EnforceMedicalCertCompliance         bool                       `json:"enforceMedicalCertCompliance"         bun:"enforce_medical_cert_compliance,type:BOOLEAN,notnull,default:true"`
	EnforceHazmatCompliance              bool                       `json:"enforceHazmatCompliance"              bun:"enforce_hazmat_compliance,type:BOOLEAN,notnull,default:true"`
	EnforceDrugAndAlcoholCompliance      bool                       `json:"enforceDrugAndAlcoholCompliance"      bun:"enforce_drug_and_alcohol_compliance,type:BOOLEAN,notnull,default:true"`
	AutoAssignmentStrategy               AutoAssignmentStrategy     `json:"autoAssignmentStrategy"               bun:"auto_assignment_strategy,type:auto_assignment_strategy_enum,notnull,default:'Proximity'"`
	ComplianceEnforcementLevel           ComplianceEnforcementLevel `json:"complianceEnforcementLevel"           bun:"compliance_enforcement_level,type:compliance_enforcement_level_enum,notnull,default:'Warning'"`
	RecordServiceFailures                ServiceIncidentType        `json:"recordServiceFailures"                bun:"record_service_failures,type:service_incident_type_enum,notnull,default:'Never'"`
	ServiceFailureGracePeriod            *int16                     `json:"serviceFailureGracePeriod"            bun:"service_failure_grace_period,type:INTEGER,nullzero"` // In minutes
	ServiceFailureTarget                 *float32                   `json:"serviceFailureTarget"                 bun:"service_failure_target,type:FLOAT,nullzero"`         // Percentage
	Version                              int64                      `json:"version"                              bun:"version,type:BIGINT"`
	CreatedAt                            int64                      `json:"createdAt"                            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                            int64                      `json:"updatedAt"                            bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (dc *DispatchControl) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(dc,
		validation.Field(&dc.ServiceFailureGracePeriod,
			validation.When(dc.RecordServiceFailures.NotEqual(ServiceIncidentTypeNever),
				validation.Required.Error("Service failure grace period is required"),
				validation.Min(1).Error("Service failure grace period must be greater than 0"),
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

func (dc *DispatchControl) GetID() string {
	return dc.ID.String()
}

func (dc *DispatchControl) GetTableName() string {
	return "dispatch_controls"
}

func (dc *DispatchControl) GetOrganizationID() pulid.ID {
	return dc.OrganizationID
}

func (dc *DispatchControl) GetBusinessUnitID() pulid.ID {
	return dc.BusinessUnitID
}

func (dc *DispatchControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dc.ID.IsNil() {
			dc.ID = pulid.MustNew("dc_")
		}

		dc.CreatedAt = now
	case *bun.UpdateQuery:
		dc.UpdatedAt = now
	}

	return nil
}

package dispatchcontrol

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*DispatchControl)(nil)
	_ validationframework.TenantedEntity = (*DispatchControl)(nil)
)

type DispatchControl struct {
	bun.BaseModel `bun:"table:dispatch_controls,alias:dc" json:"-"`

	ID                                   pulid.ID                   `json:"id"                                   bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID                       pulid.ID                   `json:"businessUnitId"                       bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID                       pulid.ID                   `json:"organizationId"                       bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	EnableAutoAssignment                 bool                       `json:"enableAutoAssignment"                 bun:"enable_auto_assignment,type:BOOLEAN,notnull,default:false"`
	AutoAssignmentStrategy               AutoAssignmentStrategy     `json:"autoAssignmentStrategy"               bun:"auto_assignment_strategy,type:auto_assignment_strategy_enum,notnull,default:'Proximity'"`
	EnforceWorkerAssign                  bool                       `json:"enforceWorkerAssign"                  bun:"enforce_worker_assign,type:BOOLEAN,notnull,default:false"`
	EnforceTrailerContinuity             bool                       `json:"enforceTrailerContinuity"             bun:"enforce_trailer_continuity,type:BOOLEAN,notnull,default:false"`
	EnforceHOSCompliance                 bool                       `json:"enforceHosCompliance"                 bun:"enforce_hos_compliance,type:BOOLEAN,notnull,default:false"`
	EnforceWorkerPTARestrictions         bool                       `json:"enforceWorkerPtaRestrictions"         bun:"enforce_worker_pta_restrictions,type:BOOLEAN,notnull,default:false"`
	EnforceWorkerTractorFleetContinuity  bool                       `json:"enforceWorkerTractorFleetContinuity"  bun:"enforce_worker_tractor_fleet_continuity,type:BOOLEAN,notnull,default:false"`
	EnforceDriverQualificationCompliance bool                       `json:"enforceDriverQualificationCompliance" bun:"enforce_driver_qualification_compliance,type:BOOLEAN,notnull,default:false"`
	EnforceMedicalCertCompliance         bool                       `json:"enforceMedicalCertCompliance"         bun:"enforce_medical_cert_compliance,type:BOOLEAN,notnull,default:false"`
	EnforceHazmatCompliance              bool                       `json:"enforceHazmatCompliance"              bun:"enforce_hazmat_compliance,type:BOOLEAN,notnull,default:false"`
	EnforceDrugAndAlcoholCompliance      bool                       `json:"enforceDrugAndAlcoholCompliance"      bun:"enforce_drug_and_alcohol_compliance,type:BOOLEAN,notnull,default:false"`
	ComplianceEnforcementLevel           ComplianceEnforcementLevel `json:"complianceEnforcementLevel"           bun:"compliance_enforcement_level,type:compliance_enforcement_level_enum,notnull,default:'Warning'"`
	RecordServiceFailures                ServiceIncidentType        `json:"recordServiceFailures"                bun:"record_service_failures,type:service_incident_type_enum,notnull,default:'Never'"`
	ServiceFailureTarget                 *float64                   `json:"serviceFailureTarget"                 bun:"service_failure_target,type:FLOAT,nullzero"`
	ServiceFailureGracePeriod            *int                       `json:"serviceFailureGracePeriod"            bun:"service_failure_grace_period,type:INTEGER,nullzero,default:30"`
	Version                              int64                      `json:"version"                              bun:"version,type:BIGINT"`
	CreatedAt                            int64                      `json:"createdAt"                            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                            int64                      `json:"updatedAt"                            bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (dc *DispatchControl) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(dc,
		validation.Field(&dc.AutoAssignmentStrategy,
			validation.Required.Error("Auto assignment strategy is required"),
			validation.By(func(value any) error {
				a, ok := value.(AutoAssignmentStrategy)
				if !ok {
					return errors.New("invalid auto assignment strategy type")
				}
				if !a.IsValid() {
					return errors.New(
						"auto assignment strategy must be one of: Proximity, Availability, LoadBalancing",
					)
				}
				return nil
			}),
		),
		validation.Field(&dc.ComplianceEnforcementLevel,
			validation.Required.Error("Compliance enforcement level is required"),
			validation.By(func(value any) error {
				c, ok := value.(ComplianceEnforcementLevel)
				if !ok {
					return errors.New("invalid compliance enforcement level type")
				}
				if !c.IsValid() {
					return errors.New(
						"compliance enforcement level must be one of: Warning, Block, Audit",
					)
				}
				return nil
			}),
		),
		validation.Field(&dc.RecordServiceFailures,
			validation.Required.Error("Record service failures is required"),
			validation.By(func(value any) error {
				s, ok := value.(ServiceIncidentType)
				if !ok {
					return errors.New("invalid service incident type")
				}
				if !s.IsValid() {
					return errors.New(
						"record service failures must be one of: Never, Pickup, Delivery, PickupDelivery, AllExceptShipper",
					)
				}
				return nil
			}),
		),
		validation.Field(&dc.ServiceFailureGracePeriod,
			validation.When(dc.RecordServiceFailures != ServiceIncidentTypeNever,
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

func (dc *DispatchControl) GetTableName() string {
	return "dispatch_controls"
}

func (dc *DispatchControl) GetID() pulid.ID {
	return dc.ID
}

func (dc *DispatchControl) GetOrganizationID() pulid.ID {
	return dc.OrganizationID
}

func (dc *DispatchControl) GetBusinessUnitID() pulid.ID {
	return dc.BusinessUnitID
}

func (dc *DispatchControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

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

func NewDefaultDispatchControl(orgID, buID pulid.ID) *DispatchControl {
	return &DispatchControl{
		OrganizationID:                       orgID,
		BusinessUnitID:                       buID,
		EnableAutoAssignment:                 true,
		AutoAssignmentStrategy:               AutoAssignmentStrategyProximity,
		EnforceWorkerAssign:                  true,
		EnforceTrailerContinuity:             true,
		EnforceHOSCompliance:                 true,
		EnforceWorkerPTARestrictions:         true,
		EnforceWorkerTractorFleetContinuity:  true,
		EnforceDriverQualificationCompliance: true,
		EnforceMedicalCertCompliance:         true,
		EnforceHazmatCompliance:              true,
		EnforceDrugAndAlcoholCompliance:      true,
		ComplianceEnforcementLevel:           ComplianceEnforcementLevelWarning,
		RecordServiceFailures:                ServiceIncidentTypeNever,
	}
}

package dispatchcontrol

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	DefaultAutoAssignPlanningHorizonHours = int16(48)
	MaxAutoAssignPlanningHorizonHours     = int16(336)

	DefaultHorizonMaxMovesPerDriver = int16(3)
	MaxHorizonMovesPerDriver        = int16(20)
)

var defaultAutoAssignConfidenceThreshold = decimal.NewFromFloat(0.85)

var (
	_ bun.BeforeAppendModelHook          = (*DispatchControl)(nil)
	_ validationframework.TenantedEntity = (*DispatchControl)(nil)
)

const DefaultServiceFailureGracePeriod = 30

type DispatchControl struct {
	bun.BaseModel `bun:"table:dispatch_controls,alias:dc" json:"-"`

	ID                                   pulid.ID                   `json:"id"                                   bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID                       pulid.ID                   `json:"businessUnitId"                       bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID                       pulid.ID                   `json:"organizationId"                       bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	EnableAutoAssignment                 bool                       `json:"enableAutoAssignment"                 bun:"enable_auto_assignment,type:BOOLEAN,notnull"`
	AutoAssignmentStrategy               AutoAssignmentStrategy     `json:"autoAssignmentStrategy"               bun:"auto_assignment_strategy,type:auto_assignment_strategy_enum,notnull,default:'Proximity'"`
	EnforceWorkerAssign                  bool                       `json:"enforceWorkerAssign"                  bun:"enforce_worker_assign,type:BOOLEAN,notnull"`
	EnforceTrailerContinuity             bool                       `json:"enforceTrailerContinuity"             bun:"enforce_trailer_continuity,type:BOOLEAN,notnull"`
	EnforceHOSCompliance                 bool                       `json:"enforceHosCompliance"                 bun:"enforce_hos_compliance,type:BOOLEAN,notnull"`
	EnforceWorkerPTARestrictions         bool                       `json:"enforceWorkerPtaRestrictions"         bun:"enforce_worker_pta_restrictions,type:BOOLEAN,notnull"`
	EnforceWorkerTractorFleetContinuity  bool                       `json:"enforceWorkerTractorFleetContinuity"  bun:"enforce_worker_tractor_fleet_continuity,type:BOOLEAN,notnull"`
	EnforceDriverQualificationCompliance bool                       `json:"enforceDriverQualificationCompliance" bun:"enforce_driver_qualification_compliance,type:BOOLEAN,notnull"`
	EnforceMedicalCertCompliance         bool                       `json:"enforceMedicalCertCompliance"         bun:"enforce_medical_cert_compliance,type:BOOLEAN,notnull"`
	EnforceHazmatCompliance              bool                       `json:"enforceHazmatCompliance"              bun:"enforce_hazmat_compliance,type:BOOLEAN,notnull"`
	EnforceDrugAndAlcoholCompliance      bool                       `json:"enforceDrugAndAlcoholCompliance"      bun:"enforce_drug_and_alcohol_compliance,type:BOOLEAN,notnull"`
	EnableAutoStopActuals                bool                       `json:"enableAutoStopActuals"                bun:"enable_auto_stop_actuals,type:BOOLEAN,notnull"`
	ScoringWeights                       ScoringWeights             `json:"scoringWeights"                       bun:"scoring_weights,type:JSONB,notnull,default:'{}'"`
	AutoAssignConfidenceThreshold        decimal.Decimal            `json:"autoAssignConfidenceThreshold"        bun:"auto_assign_confidence_threshold,type:NUMERIC(5,4),notnull,default:0.85"`
	AutoAssignMaxDeadheadMiles           *int32                     `json:"autoAssignMaxDeadheadMiles"           bun:"auto_assign_max_deadhead_miles,type:INTEGER,nullzero"`
	AutoAssignPlanningHorizonHours       int16                      `json:"autoAssignPlanningHorizonHours"       bun:"auto_assign_planning_horizon_hours,type:SMALLINT,notnull,default:48"`
	PlanningMode                         PlanningMode               `json:"planningMode"                         bun:"planning_mode,type:dispatch_planning_mode_enum,notnull,default:'Immediate'"`
	HorizonMaxMovesPerDriver             int16                      `json:"horizonMaxMovesPerDriver"             bun:"horizon_max_moves_per_driver,type:SMALLINT,notnull,default:3"`
	ComplianceEnforcementLevel           ComplianceEnforcementLevel `json:"complianceEnforcementLevel"           bun:"compliance_enforcement_level,type:compliance_enforcement_level_enum,notnull,default:'Warning'"`
	RecordServiceFailures                ServiceIncidentType        `json:"recordServiceFailures"                bun:"record_service_failures,type:service_incident_type_enum,notnull,default:'Never'"`
	ServiceFailureTarget                 *float64                   `json:"serviceFailureTarget"                 bun:"service_failure_target,type:FLOAT,nullzero"`
	ServiceFailureGracePeriod            *int                       `json:"serviceFailureGracePeriod"            bun:"service_failure_grace_period,type:INTEGER,nullzero,default:30"`
	Version                              int64                      `json:"version"                              bun:"version,type:BIGINT"`
	CreatedAt                            int64                      `json:"createdAt"                            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                            int64                      `json:"updatedAt"                            bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

// ResolvedScoringWeights is the weighting the candidate scorer should use for this
// organization: the strategy preset with any stored per-factor overrides applied.
func (dc *DispatchControl) ResolvedScoringWeights() map[ScoringFactor]float64 {
	if dc == nil {
		return PresetWeights(AutoAssignmentStrategyProximity)
	}
	return dc.ScoringWeights.Resolve(dc.AutoAssignmentStrategy)
}

// PlanningHorizonHours is the forward window the console and the optimizer plan over,
// falling back to the default when an organization has never configured one.
func (dc *DispatchControl) PlanningHorizonHours() int16 {
	if dc == nil || dc.AutoAssignPlanningHorizonHours <= 0 {
		return DefaultAutoAssignPlanningHorizonHours
	}
	if dc.AutoAssignPlanningHorizonHours > MaxAutoAssignPlanningHorizonHours {
		return MaxAutoAssignPlanningHorizonHours
	}
	return dc.AutoAssignPlanningHorizonHours
}

// ResolvedPlanningMode falls back to Immediate so controls written before horizon
// planning existed keep the single-period behaviour they were configured against.
func (dc *DispatchControl) ResolvedPlanningMode() PlanningMode {
	if dc == nil || !dc.PlanningMode.IsValid() {
		return PlanningModeImmediate
	}
	return dc.PlanningMode
}

// HorizonMovesPerDriver caps how many moves horizon planning may chain onto one
// driver, keeping a single driver from absorbing the whole board.
func (dc *DispatchControl) HorizonMovesPerDriver() int {
	if dc == nil || dc.HorizonMaxMovesPerDriver <= 0 {
		return int(DefaultHorizonMaxMovesPerDriver)
	}
	if dc.HorizonMaxMovesPerDriver > MaxHorizonMovesPerDriver {
		return int(MaxHorizonMovesPerDriver)
	}
	return int(dc.HorizonMaxMovesPerDriver)
}

// ConfidenceThreshold is the minimum confidence an auto-assign proposal must carry
// before it may execute without a dispatcher reviewing it.
func (dc *DispatchControl) ConfidenceThreshold() decimal.Decimal {
	if dc == nil || dc.AutoAssignConfidenceThreshold.IsZero() {
		return defaultAutoAssignConfidenceThreshold
	}
	return dc.AutoAssignConfidenceThreshold
}

func (dc *DispatchControl) NormalizeServiceFailureSettings() {
	if dc == nil {
		return
	}
	if !dc.RecordServiceFailures.IsValid() ||
		dc.RecordServiceFailures == ServiceIncidentTypeNever {
		return
	}
	if dc.ServiceFailureGracePeriod != nil {
		return
	}

	defaultGracePeriod := DefaultServiceFailureGracePeriod
	dc.ServiceFailureGracePeriod = &defaultGracePeriod
}

// NormalizeAutoAssignmentSettings fills in the auto-assignment defaults for controls
// that predate them or were constructed in memory, so a zero value is treated as
// "unset" rather than as an invalid configuration.
func (dc *DispatchControl) NormalizeAutoAssignmentSettings() {
	if dc == nil {
		return
	}

	if dc.ScoringWeights == nil {
		dc.ScoringWeights = ScoringWeights{}
	}
	if dc.AutoAssignConfidenceThreshold.IsZero() {
		dc.AutoAssignConfidenceThreshold = defaultAutoAssignConfidenceThreshold
	}
	if dc.AutoAssignPlanningHorizonHours <= 0 {
		dc.AutoAssignPlanningHorizonHours = DefaultAutoAssignPlanningHorizonHours
	}
	if !dc.PlanningMode.IsValid() {
		dc.PlanningMode = PlanningModeImmediate
	}
	if dc.HorizonMaxMovesPerDriver <= 0 {
		dc.HorizonMaxMovesPerDriver = DefaultHorizonMaxMovesPerDriver
	}
}

func (dc *DispatchControl) Validate(multiErr *errortypes.MultiError) {
	dc.NormalizeServiceFailureSettings()
	dc.NormalizeAutoAssignmentSettings()

	multiErr.AddOzzoError(validation.ValidateStruct(dc,
		validation.Field(
			&dc.AutoAssignmentStrategy,
			validation.Required.Error("Auto assignment strategy is required"),
			domainvalidation.ValidEnum[AutoAssignmentStrategy](
				"auto assignment strategy must be one of: Proximity, Availability, LoadBalancing, Performance",
			),
		),
		validation.Field(
			&dc.PlanningMode,
			validation.Required.Error("Planning mode is required"),
			domainvalidation.ValidEnum[PlanningMode](
				"planning mode must be one of: Immediate, Horizon",
			),
		),
		validation.Field(&dc.HorizonMaxMovesPerDriver,
			validation.Min(int16(1)).Error(
				"Horizon max moves per driver must be greater than 0",
			),
			validation.Max(MaxHorizonMovesPerDriver).Error(
				"Horizon max moves per driver must be 20 or fewer",
			),
		),
		validation.Field(
			&dc.ComplianceEnforcementLevel,
			validation.Required.Error("Compliance enforcement level is required"),
			domainvalidation.ValidEnum[ComplianceEnforcementLevel](
				"compliance enforcement level must be one of: Warning, Block, Audit",
			),
		),
		validation.Field(
			&dc.RecordServiceFailures,
			validation.Required.Error("Record service failures is required"),
			domainvalidation.ValidEnum[ServiceIncidentType](
				"record service failures must be one of: Never, Pickup, Delivery, PickupDelivery, AllExceptShipper",
			),
		),
		validation.Field(&dc.ServiceFailureGracePeriod,
			validation.When(dc.ServiceFailureGracePeriod != nil,
				validation.By(func(_ any) error {
					if *dc.ServiceFailureGracePeriod <= 0 {
						return errors.New("Service failure grace period must be greater than 0")
					}
					return nil
				}),
			),
		),
		validation.Field(&dc.ScoringWeights,
			validation.By(func(_ any) error {
				if err := dc.ScoringWeights.Validate(); err != nil {
					return errors.New(
						"Scoring weights must reference known factors with values between 0 and 10",
					)
				}
				return nil
			}),
		),
		validation.Field(&dc.AutoAssignConfidenceThreshold,
			validation.By(func(_ any) error {
				if dc.AutoAssignConfidenceThreshold.LessThan(decimal.Zero) ||
					dc.AutoAssignConfidenceThreshold.GreaterThan(decimal.NewFromInt(1)) {
					return errors.New("Auto assign confidence threshold must be between 0 and 1")
				}
				return nil
			}),
		),
		validation.Field(&dc.AutoAssignMaxDeadheadMiles,
			validation.When(dc.AutoAssignMaxDeadheadMiles != nil,
				validation.By(func(_ any) error {
					if *dc.AutoAssignMaxDeadheadMiles <= 0 {
						return errors.New("Auto assign max deadhead miles must be greater than 0")
					}
					return nil
				}),
			),
		),
		validation.Field(&dc.AutoAssignPlanningHorizonHours,
			validation.By(func(_ any) error {
				if dc.AutoAssignPlanningHorizonHours <= 0 ||
					dc.AutoAssignPlanningHorizonHours > MaxAutoAssignPlanningHorizonHours {
					return errors.New(
						"Auto assign planning horizon must be between 1 and 336 hours",
					)
				}
				return nil
			}),
		),
	))
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
		ScoringWeights:                       ScoringWeights{},
		AutoAssignConfidenceThreshold:        defaultAutoAssignConfidenceThreshold,
		AutoAssignPlanningHorizonHours:       DefaultAutoAssignPlanningHorizonHours,
	}
}

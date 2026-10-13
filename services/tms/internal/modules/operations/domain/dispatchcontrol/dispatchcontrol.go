package dispatchcontrol

import (
	"context"
	"errors"
	"time"

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

	DefaultHorizonSearchIterations = int16(25)
	MaxHorizonSearchIterations     = int16(500)

	// A move that still has nobody on it this close to its start is raised to
	// the desk, and to whichever agent covers dispatch. The window is shorter
	// than the planning horizon on purpose: the planner looks ahead so it can
	// build a good day, while this asks who is going to run a load that is
	// about to leave.
	DefaultCoverageRiskWindowHours = int16(12)
	MaxCoverageRiskWindowHours     = int16(168)

	DefaultCheckCallIntervalHours = int16(4)
	MinCheckCallIntervalHours     = int16(1)
	MaxCheckCallIntervalHours     = int16(72)

	DefaultEtaRiskThresholdMinutes = int16(60)
	MaxEtaRiskThresholdMinutes     = int16(1440)
	DefaultEtaDwellMinutes         = int16(60)
	MaxEtaDwellMinutes             = int16(720)
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
	HorizonSearchIterations              *int16                     `json:"horizonSearchIterations"              bun:"horizon_search_iterations,type:SMALLINT,nullzero"`
	CoverageRiskWindowHours              int16                      `json:"coverageRiskWindowHours"              bun:"coverage_risk_window_hours,type:SMALLINT,notnull,default:12"`
	EnableCheckCallMonitoring            bool                       `json:"enableCheckCallMonitoring"            bun:"enable_check_call_monitoring,type:BOOLEAN,notnull,default:false"`
	CheckCallIntervalHours               int16                      `json:"checkCallIntervalHours"               bun:"check_call_interval_hours,type:SMALLINT,notnull,default:4"`
	EnableEtaAlerts                      bool                       `json:"enableEtaAlerts"                      bun:"enable_eta_alerts,type:BOOLEAN,notnull,default:true"`
	EtaRiskThresholdMinutes              int16                      `json:"etaRiskThresholdMinutes"              bun:"eta_risk_threshold_minutes,type:SMALLINT,notnull,default:60"`
	EtaDwellMinutes                      int16                      `json:"etaDwellMinutes"                      bun:"eta_dwell_minutes,type:SMALLINT,notnull,default:60"`
	ComplianceEnforcementLevel           ComplianceEnforcementLevel `json:"complianceEnforcementLevel"           bun:"compliance_enforcement_level,type:compliance_enforcement_level_enum,notnull,default:'Warning'"`
	RecordServiceFailures                ServiceIncidentType        `json:"recordServiceFailures"                bun:"record_service_failures,type:service_incident_type_enum,notnull,default:'Never'"`
	ServiceFailureTarget                 *float64                   `json:"serviceFailureTarget"                 bun:"service_failure_target,type:FLOAT,nullzero"`
	ServiceFailureGracePeriod            *int                       `json:"serviceFailureGracePeriod"            bun:"service_failure_grace_period,type:INTEGER,nullzero,default:30"`
	Version                              int64                      `json:"version"                              bun:"version,type:BIGINT"`
	CreatedAt                            int64                      `json:"createdAt"                            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                            int64                      `json:"updatedAt"                            bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (dc *DispatchControl) ResolvedScoringWeights() map[ScoringFactor]float64 {
	if dc == nil {
		return PresetWeights(AutoAssignmentStrategyProximity)
	}
	return dc.ScoringWeights.Resolve(dc.AutoAssignmentStrategy)
}

func (dc *DispatchControl) PlanningHorizonHours() int16 {
	if dc == nil || dc.AutoAssignPlanningHorizonHours <= 0 {
		return DefaultAutoAssignPlanningHorizonHours
	}
	if dc.AutoAssignPlanningHorizonHours > MaxAutoAssignPlanningHorizonHours {
		return MaxAutoAssignPlanningHorizonHours
	}
	return dc.AutoAssignPlanningHorizonHours
}

func (dc *DispatchControl) CoverageWindowHours() int16 {
	if dc == nil || dc.CoverageRiskWindowHours <= 0 {
		return DefaultCoverageRiskWindowHours
	}
	if dc.CoverageRiskWindowHours > MaxCoverageRiskWindowHours {
		return MaxCoverageRiskWindowHours
	}

	return dc.CoverageRiskWindowHours
}

func (dc *DispatchControl) CheckCallWindowHours(customerHours *int16) int16 {
	if customerHours != nil && validCheckCallInterval(*customerHours) {
		return *customerHours
	}
	if dc == nil || !validCheckCallInterval(dc.CheckCallIntervalHours) {
		return DefaultCheckCallIntervalHours
	}
	return dc.CheckCallIntervalHours
}

func validCheckCallInterval(hours int16) bool {
	return hours >= MinCheckCallIntervalHours && hours <= MaxCheckCallIntervalHours
}

func (dc *DispatchControl) EtaThresholdMinutes(customerMinutes *int16) int16 {
	if customerMinutes != nil && validEtaRiskThreshold(*customerMinutes) {
		return *customerMinutes
	}
	if dc == nil || !validEtaRiskThreshold(dc.EtaRiskThresholdMinutes) {
		return DefaultEtaRiskThresholdMinutes
	}
	return dc.EtaRiskThresholdMinutes
}

func (dc *DispatchControl) EtaDwell() int16 {
	if dc == nil || !validEtaDwell(dc.EtaDwellMinutes) {
		return DefaultEtaDwellMinutes
	}
	return dc.EtaDwellMinutes
}

func (dc *DispatchControl) EtaAlertsEnabled() bool {
	return dc == nil || dc.EnableEtaAlerts
}

func validEtaRiskThreshold(minutes int16) bool {
	return minutes >= 0 && minutes <= MaxEtaRiskThresholdMinutes
}

func validEtaDwell(minutes int16) bool {
	return minutes >= 0 && minutes <= MaxEtaDwellMinutes
}

func (dc *DispatchControl) StartsInsideCoverageWindow(startsAt, now int64) bool {
	return startsAt <= now+int64(dc.CoverageWindowHours())*int64(time.Hour/time.Second)
}

func (dc *DispatchControl) ResolvedPlanningMode() PlanningMode {
	if dc == nil || !dc.PlanningMode.IsValid() {
		return PlanningModeImmediate
	}
	return dc.PlanningMode
}

func (dc *DispatchControl) HorizonMovesPerDriver() int {
	if dc == nil || dc.HorizonMaxMovesPerDriver <= 0 {
		return int(DefaultHorizonMaxMovesPerDriver)
	}
	if dc.HorizonMaxMovesPerDriver > MaxHorizonMovesPerDriver {
		return int(MaxHorizonMovesPerDriver)
	}
	return int(dc.HorizonMaxMovesPerDriver)
}

func (dc *DispatchControl) HorizonSearchRounds() int {
	if dc == nil || dc.HorizonSearchIterations == nil {
		return int(DefaultHorizonSearchIterations)
	}
	if *dc.HorizonSearchIterations == 0 {
		return 0
	}
	if *dc.HorizonSearchIterations < 0 {
		return int(DefaultHorizonSearchIterations)
	}

	return min(int(*dc.HorizonSearchIterations), int(MaxHorizonSearchIterations))
}

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
	if dc.CoverageRiskWindowHours <= 0 {
		dc.CoverageRiskWindowHours = DefaultCoverageRiskWindowHours
	}
	if dc.CheckCallIntervalHours <= 0 {
		dc.CheckCallIntervalHours = DefaultCheckCallIntervalHours
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
		validation.Field(&dc.HorizonSearchIterations,
			validation.Min(int16(0)).Error(
				"Horizon search iterations cannot be negative",
			),
			validation.Max(MaxHorizonSearchIterations).Error(
				"Horizon search iterations must be 500 or fewer",
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
		validation.Field(&dc.EtaRiskThresholdMinutes,
			validation.By(func(_ any) error {
				if !validEtaRiskThreshold(dc.EtaRiskThresholdMinutes) {
					return errors.New("ETA risk threshold must be between 0 and 1440 minutes")
				}
				return nil
			}),
		),
		validation.Field(&dc.EtaDwellMinutes,
			validation.By(func(_ any) error {
				if !validEtaDwell(dc.EtaDwellMinutes) {
					return errors.New("Stop dwell time must be between 0 and 720 minutes")
				}
				return nil
			}),
		),
		validation.Field(&dc.CheckCallIntervalHours,
			validation.By(func(_ any) error {
				if !validCheckCallInterval(dc.CheckCallIntervalHours) {
					return errors.New("Check call interval must be between 1 and 72 hours")
				}
				return nil
			}),
		),
		validation.Field(&dc.CoverageRiskWindowHours,
			validation.By(func(_ any) error {
				if dc.CoverageRiskWindowHours <= 0 ||
					dc.CoverageRiskWindowHours > MaxCoverageRiskWindowHours {
					return errors.New(
						"Coverage risk window must be between 1 and 168 hours",
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
		CoverageRiskWindowHours:              DefaultCoverageRiskWindowHours,
		EnableEtaAlerts:                      true,
		EtaRiskThresholdMinutes:              DefaultEtaRiskThresholdMinutes,
		EtaDwellMinutes:                      DefaultEtaDwellMinutes,
	}
}

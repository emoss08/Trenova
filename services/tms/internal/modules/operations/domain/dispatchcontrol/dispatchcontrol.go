package dispatchcontrol

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
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

	DefaultExceptionDueInfoMinutes         = int32(1440)
	DefaultExceptionDueWarningMinutes      = int32(240)
	DefaultExceptionDueCriticalMinutes     = int32(60)
	DefaultExceptionEscalationMinutes      = int32(60)
	DefaultExceptionMaxEscalations         = int16(3)
	MinExceptionDueMinutes                 = int32(5)
	MaxExceptionDueMinutes                 = int32(10080)
	MinExceptionEscalationMinutes          = int32(5)
	MaxExceptionEscalationMinutes          = int32(1440)
	MaxExceptionEscalations                = int16(10)
	DefaultExceptionMissedStopGraceMinutes = int16(30)
	MaxExceptionMissedStopGraceMinutes     = int16(1440)
	DefaultExceptionPODGraceHours          = int16(24)
	MinExceptionPODGraceHours              = int16(1)
	MaxExceptionPODGraceHours              = int16(720)

	DefaultTenderCounterDecisionMinutes = int32(60)
	MinTenderCounterDecisionMinutes     = int32(5)
	MaxTenderCounterDecisionMinutes     = int32(1440)
	DefaultTenderEscalationTTLMinutes   = int32(60)
	MinTenderEscalationTTLMinutes       = int32(5)
	MaxTenderEscalationTTLMinutes       = int32(10080)
	DefaultTenderMaxEscalations         = int16(1)
	MaxTenderEscalations                = int16(5)
)

var (
	DefaultTenderCounterMarginFloor = decimal.NewFromInt(10)
	minTenderMarginFloor            = decimal.NewFromInt(-100)
	maxPercent                      = decimal.NewFromInt(100)
)

var defaultAutoAssignConfidenceThreshold = decimal.NewFromFloat(0.85)

var (
	_ validationframework.TenantedEntity = (*DispatchControl)(nil)
)

const DefaultServiceFailureGracePeriod = 30

type DispatchControl struct {
	keel.Table `keel:"dispatch_controls"`

	ID                                   pulid.ID                   `json:"id"                                   keel:"id,pk,autoid:dc_"`
	BusinessUnitID                       pulid.ID                   `json:"businessUnitId"                       keel:"business_unit_id,pk,tenant"`
	OrganizationID                       pulid.ID                   `json:"organizationId"                       keel:"organization_id,pk,tenant"`
	EnableAutoAssignment                 bool                       `json:"enableAutoAssignment"                 keel:"enable_auto_assignment"`
	AutoAssignmentStrategy               AutoAssignmentStrategy     `json:"autoAssignmentStrategy"               keel:"auto_assignment_strategy"`
	EnforceWorkerAssign                  bool                       `json:"enforceWorkerAssign"                  keel:"enforce_worker_assign"`
	EnforceTrailerContinuity             bool                       `json:"enforceTrailerContinuity"             keel:"enforce_trailer_continuity"`
	EnforceHOSCompliance                 bool                       `json:"enforceHosCompliance"                 keel:"enforce_hos_compliance"`
	EnforceWorkerPTARestrictions         bool                       `json:"enforceWorkerPtaRestrictions"         keel:"enforce_worker_pta_restrictions"`
	EnforceWorkerTractorFleetContinuity  bool                       `json:"enforceWorkerTractorFleetContinuity"  keel:"enforce_worker_tractor_fleet_continuity"`
	EnforceDriverQualificationCompliance bool                       `json:"enforceDriverQualificationCompliance" keel:"enforce_driver_qualification_compliance"`
	EnforceMedicalCertCompliance         bool                       `json:"enforceMedicalCertCompliance"         keel:"enforce_medical_cert_compliance"`
	EnforceHazmatCompliance              bool                       `json:"enforceHazmatCompliance"              keel:"enforce_hazmat_compliance"`
	EnforceDrugAndAlcoholCompliance      bool                       `json:"enforceDrugAndAlcoholCompliance"      keel:"enforce_drug_and_alcohol_compliance"`
	EnableAutoStopActuals                bool                       `json:"enableAutoStopActuals"                keel:"enable_auto_stop_actuals"`
	ScoringWeights                       ScoringWeights             `json:"scoringWeights"                       keel:"scoring_weights,jsonb"`
	AutoAssignConfidenceThreshold        decimal.Decimal            `json:"autoAssignConfidenceThreshold"        keel:"auto_assign_confidence_threshold"`
	AutoAssignMaxDeadheadMiles           *int32                     `json:"autoAssignMaxDeadheadMiles"           keel:"auto_assign_max_deadhead_miles"`
	AutoAssignPlanningHorizonHours       int16                      `json:"autoAssignPlanningHorizonHours"       keel:"auto_assign_planning_horizon_hours"`
	PlanningMode                         PlanningMode               `json:"planningMode"                         keel:"planning_mode"`
	HorizonMaxMovesPerDriver             int16                      `json:"horizonMaxMovesPerDriver"             keel:"horizon_max_moves_per_driver"`
	HorizonSearchIterations              *int16                     `json:"horizonSearchIterations"              keel:"horizon_search_iterations"`
	CoverageRiskWindowHours              int16                      `json:"coverageRiskWindowHours"              keel:"coverage_risk_window_hours"`
	EnableCheckCallMonitoring            bool                       `json:"enableCheckCallMonitoring"            keel:"enable_check_call_monitoring"`
	CheckCallIntervalHours               int16                      `json:"checkCallIntervalHours"               keel:"check_call_interval_hours"`
	EnableEtaAlerts                      bool                       `json:"enableEtaAlerts"                      keel:"enable_eta_alerts"`
	EtaRiskThresholdMinutes              int16                      `json:"etaRiskThresholdMinutes"              keel:"eta_risk_threshold_minutes"`
	EtaDwellMinutes                      int16                      `json:"etaDwellMinutes"                      keel:"eta_dwell_minutes"`
	ExceptionDueInfoMinutes              int32                      `json:"exceptionDueInfoMinutes"              keel:"exception_due_info_minutes"`
	ExceptionDueWarningMinutes           int32                      `json:"exceptionDueWarningMinutes"           keel:"exception_due_warning_minutes"`
	ExceptionDueCriticalMinutes          int32                      `json:"exceptionDueCriticalMinutes"          keel:"exception_due_critical_minutes"`
	ExceptionEscalationMinutes           int32                      `json:"exceptionEscalationMinutes"           keel:"exception_escalation_minutes"`
	ExceptionMaxEscalations              int16                      `json:"exceptionMaxEscalations"              keel:"exception_max_escalations"`
	ExceptionDefaultAssigneeID           pulid.ID                   `json:"exceptionDefaultAssigneeId"           keel:"exception_default_assignee_id,nullzero"`
	ExceptionEscalationContactID         pulid.ID                   `json:"exceptionEscalationContactId"         keel:"exception_escalation_contact_id,nullzero"`
	ExceptionMissedStopGraceMinutes      int16                      `json:"exceptionMissedStopGraceMinutes"      keel:"exception_missed_stop_grace_minutes"`
	ExceptionPODGraceHours               int16                      `json:"exceptionPodGraceHours"               keel:"exception_pod_grace_hours"`
	TenderCounterOffersEnabled           bool                       `json:"tenderCounterOffersEnabled"           keel:"tender_counter_offers_enabled"`
	TenderCounterMarginFloorPercent      decimal.NullDecimal        `json:"tenderCounterMarginFloorPercent"      keel:"tender_counter_margin_floor_percent"`
	TenderAutoAcceptCounters             bool                       `json:"tenderAutoAcceptCounters"             keel:"tender_auto_accept_counters"`
	TenderCounterDecisionMinutes         int32                      `json:"tenderCounterDecisionMinutes"         keel:"tender_counter_decision_minutes"`
	TenderExhaustedAction                TenderExhaustedAction      `json:"tenderExhaustedAction"                keel:"tender_exhausted_action"`
	TenderEscalationRateIncreasePercent  decimal.Decimal            `json:"tenderEscalationRateIncreasePercent"  keel:"tender_escalation_rate_increase_percent"`
	TenderEscalationOfferTTLMinutes      int32                      `json:"tenderEscalationOfferTtlMinutes"      keel:"tender_escalation_offer_ttl_minutes"`
	TenderMaxEscalations                 int16                      `json:"tenderMaxEscalations"                 keel:"tender_max_escalations"`
	ComplianceEnforcementLevel           ComplianceEnforcementLevel `json:"complianceEnforcementLevel"           keel:"compliance_enforcement_level"`
	RecordServiceFailures                ServiceIncidentType        `json:"recordServiceFailures"                keel:"record_service_failures"`
	ServiceFailureTarget                 *float64                   `json:"serviceFailureTarget"                 keel:"service_failure_target"`
	ServiceFailureGracePeriod            *int                       `json:"serviceFailureGracePeriod"            keel:"service_failure_grace_period"`
	Version                              int64                      `json:"version"                              keel:"version,version"`
	CreatedAt                            int64                      `json:"createdAt"                            keel:"created_at,autocreate"`
	UpdatedAt                            int64                      `json:"updatedAt"                            keel:"updated_at,autoupdate"`
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

type ExceptionPolicy struct {
	DueInfoMinutes     int32
	DueWarningMinutes  int32
	DueCriticalMinutes int32
	EscalationMinutes  int32
	MaxEscalations     int16
	DefaultAssigneeID  pulid.ID
	EscalationContact  pulid.ID
	MissedStopGrace    int16
	PODGraceHours      int16
}

func (dc *DispatchControl) ExceptionPolicy() ExceptionPolicy {
	policy := ExceptionPolicy{
		DueInfoMinutes:     DefaultExceptionDueInfoMinutes,
		DueWarningMinutes:  DefaultExceptionDueWarningMinutes,
		DueCriticalMinutes: DefaultExceptionDueCriticalMinutes,
		EscalationMinutes:  DefaultExceptionEscalationMinutes,
		MaxEscalations:     DefaultExceptionMaxEscalations,
		MissedStopGrace:    DefaultExceptionMissedStopGraceMinutes,
		PODGraceHours:      DefaultExceptionPODGraceHours,
	}
	if dc == nil {
		return policy
	}
	if validExceptionDue(dc.ExceptionDueInfoMinutes) {
		policy.DueInfoMinutes = dc.ExceptionDueInfoMinutes
	}
	if validExceptionDue(dc.ExceptionDueWarningMinutes) {
		policy.DueWarningMinutes = dc.ExceptionDueWarningMinutes
	}
	if validExceptionDue(dc.ExceptionDueCriticalMinutes) {
		policy.DueCriticalMinutes = dc.ExceptionDueCriticalMinutes
	}
	if validExceptionEscalation(dc.ExceptionEscalationMinutes) {
		policy.EscalationMinutes = dc.ExceptionEscalationMinutes
	}
	if dc.ExceptionMaxEscalations >= 0 && dc.ExceptionMaxEscalations <= MaxExceptionEscalations {
		policy.MaxEscalations = dc.ExceptionMaxEscalations
	}
	if dc.ExceptionMissedStopGraceMinutes >= 0 &&
		dc.ExceptionMissedStopGraceMinutes <= MaxExceptionMissedStopGraceMinutes {
		policy.MissedStopGrace = dc.ExceptionMissedStopGraceMinutes
	}
	if validExceptionPODGrace(dc.ExceptionPODGraceHours) {
		policy.PODGraceHours = dc.ExceptionPODGraceHours
	}
	policy.DefaultAssigneeID = dc.ExceptionDefaultAssigneeID
	policy.EscalationContact = dc.ExceptionEscalationContactID
	return policy
}

func (dc *DispatchControl) NormalizeExceptionSettings() {
	if dc.ExceptionDueInfoMinutes == 0 {
		dc.ExceptionDueInfoMinutes = DefaultExceptionDueInfoMinutes
	}
	if dc.ExceptionDueWarningMinutes == 0 {
		dc.ExceptionDueWarningMinutes = DefaultExceptionDueWarningMinutes
	}
	if dc.ExceptionDueCriticalMinutes == 0 {
		dc.ExceptionDueCriticalMinutes = DefaultExceptionDueCriticalMinutes
	}
	if dc.ExceptionEscalationMinutes == 0 {
		dc.ExceptionEscalationMinutes = DefaultExceptionEscalationMinutes
	}
	if dc.ExceptionPODGraceHours == 0 {
		dc.ExceptionPODGraceHours = DefaultExceptionPODGraceHours
	}
}

// TenderPolicy is how an organization's tenders treat a counter-offer and an
// exhausted waterfall.
type TenderPolicy struct {
	CountersEnabled        bool
	CounterMarginFloor     decimal.Decimal
	AutoAcceptCounters     bool
	CounterDecisionSeconds int64
	ExhaustedAction        TenderExhaustedAction
	EscalationRateIncrease decimal.Decimal
	EscalationOfferTTLSecs int32
	MaxEscalations         int16
}

// DefaultTenderPolicy is the policy of an organization that has never set one:
// counters are taken for a person to decide against a 10% floor, and an
// exhausted tender only tells dispatch.
func DefaultTenderPolicy() TenderPolicy {
	return TenderPolicy{
		CountersEnabled:        true,
		CounterMarginFloor:     DefaultTenderCounterMarginFloor,
		CounterDecisionSeconds: int64(DefaultTenderCounterDecisionMinutes) * 60,
		ExhaustedAction:        TenderExhaustedActionNotify,
		EscalationRateIncrease: decimal.Zero,
		EscalationOfferTTLSecs: DefaultTenderEscalationTTLMinutes * 60,
		MaxEscalations:         DefaultTenderMaxEscalations,
	}
}

func (dc *DispatchControl) TenderPolicy() TenderPolicy {
	policy := DefaultTenderPolicy()
	if dc == nil {
		return policy
	}
	policy.CountersEnabled = dc.TenderCounterOffersEnabled
	policy.AutoAcceptCounters = dc.TenderAutoAcceptCounters
	if floor := dc.TenderCounterMarginFloorPercent; floor.Valid &&
		validTenderMarginFloor(floor.Decimal) {
		policy.CounterMarginFloor = floor.Decimal
	}
	if validTenderCounterDecision(dc.TenderCounterDecisionMinutes) {
		policy.CounterDecisionSeconds = int64(dc.TenderCounterDecisionMinutes) * 60
	}
	if dc.TenderExhaustedAction.IsValid() {
		policy.ExhaustedAction = dc.TenderExhaustedAction
	}
	if validPercent(dc.TenderEscalationRateIncreasePercent) {
		policy.EscalationRateIncrease = dc.TenderEscalationRateIncreasePercent
	}
	if validTenderEscalationTTL(dc.TenderEscalationOfferTTLMinutes) {
		policy.EscalationOfferTTLSecs = dc.TenderEscalationOfferTTLMinutes * 60
	}
	if dc.TenderMaxEscalations >= 0 && dc.TenderMaxEscalations <= MaxTenderEscalations {
		policy.MaxEscalations = dc.TenderMaxEscalations
	}
	return policy
}

func (dc *DispatchControl) NormalizeTenderSettings() {
	if dc.TenderCounterDecisionMinutes == 0 {
		dc.TenderCounterDecisionMinutes = DefaultTenderCounterDecisionMinutes
	}
	if dc.TenderEscalationOfferTTLMinutes == 0 {
		dc.TenderEscalationOfferTTLMinutes = DefaultTenderEscalationTTLMinutes
	}
	if dc.TenderExhaustedAction == "" {
		dc.TenderExhaustedAction = TenderExhaustedActionNotify
	}
	if !dc.TenderCounterMarginFloorPercent.Valid {
		dc.TenderCounterMarginFloorPercent = decimal.NewNullDecimal(
			DefaultTenderCounterMarginFloor,
		)
	}
}

func validTenderMarginFloor(floor decimal.Decimal) bool {
	return !floor.LessThan(minTenderMarginFloor) && !floor.GreaterThan(maxPercent)
}

func validPercent(value decimal.Decimal) bool {
	return !value.IsNegative() && !value.GreaterThan(maxPercent)
}

func validTenderCounterDecision(minutes int32) bool {
	return minutes >= MinTenderCounterDecisionMinutes && minutes <= MaxTenderCounterDecisionMinutes
}

func validTenderEscalationTTL(minutes int32) bool {
	return minutes >= MinTenderEscalationTTLMinutes && minutes <= MaxTenderEscalationTTLMinutes
}

func validExceptionPODGrace(hours int16) bool {
	return hours >= MinExceptionPODGraceHours && hours <= MaxExceptionPODGraceHours
}

func validExceptionDue(minutes int32) bool {
	return minutes >= MinExceptionDueMinutes && minutes <= MaxExceptionDueMinutes
}

func validExceptionEscalation(minutes int32) bool {
	return minutes >= MinExceptionEscalationMinutes && minutes <= MaxExceptionEscalationMinutes
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
	dc.NormalizeExceptionSettings()
	dc.NormalizeTenderSettings()

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
		validation.Field(&dc.ExceptionDueInfoMinutes,
			validation.By(func(_ any) error {
				if !validExceptionDue(dc.ExceptionDueInfoMinutes) {
					return errors.New("Due time must be between 5 and 10080 minutes")
				}
				return nil
			}),
		),
		validation.Field(&dc.ExceptionDueWarningMinutes,
			validation.By(func(_ any) error {
				if !validExceptionDue(dc.ExceptionDueWarningMinutes) {
					return errors.New("Due time must be between 5 and 10080 minutes")
				}
				return nil
			}),
		),
		validation.Field(&dc.ExceptionDueCriticalMinutes,
			validation.By(func(_ any) error {
				if !validExceptionDue(dc.ExceptionDueCriticalMinutes) {
					return errors.New("Due time must be between 5 and 10080 minutes")
				}
				return nil
			}),
		),
		validation.Field(&dc.ExceptionEscalationMinutes,
			validation.By(func(_ any) error {
				if !validExceptionEscalation(dc.ExceptionEscalationMinutes) {
					return errors.New("Escalation interval must be between 5 and 1440 minutes")
				}
				return nil
			}),
		),
		validation.Field(&dc.ExceptionMaxEscalations,
			validation.Min(int16(0)).Error("Escalations must be between 0 and 10"),
			validation.Max(MaxExceptionEscalations).Error("Escalations must be between 0 and 10"),
		),
		validation.Field(&dc.ExceptionMissedStopGraceMinutes,
			validation.Min(int16(0)).Error("Missed stop grace must be between 0 and 1440 minutes"),
			validation.Max(MaxExceptionMissedStopGraceMinutes).
				Error("Missed stop grace must be between 0 and 1440 minutes"),
		),
		validation.Field(&dc.ExceptionPODGraceHours,
			validation.By(func(_ any) error {
				if !validExceptionPODGrace(dc.ExceptionPODGraceHours) {
					return errors.New("POD grace must be between 1 and 720 hours")
				}
				return nil
			}),
		),
		validation.Field(&dc.TenderCounterMarginFloorPercent,
			validation.By(func(_ any) error {
				if !validTenderMarginFloor(dc.TenderCounterMarginFloorPercent.Decimal) {
					return errors.New("Counter-offer margin floor must be between -100% and 100%")
				}
				return nil
			}),
		),
		validation.Field(&dc.TenderCounterDecisionMinutes,
			validation.By(func(_ any) error {
				if !validTenderCounterDecision(dc.TenderCounterDecisionMinutes) {
					return errors.New(
						"Counter-offer decision time must be between 5 and 1440 minutes",
					)
				}
				return nil
			}),
		),
		validation.Field(&dc.TenderExhaustedAction,
			validation.Required.Error("Exhausted tender action is required"),
			domainvalidation.ValidEnum[TenderExhaustedAction](
				"Exhausted tender action must be one of: Notify, Rebroadcast, Rewaterfall",
			),
		),
		validation.Field(&dc.TenderEscalationRateIncreasePercent,
			validation.By(func(_ any) error {
				if !validPercent(dc.TenderEscalationRateIncreasePercent) {
					return errors.New("Escalation rate increase must be between 0% and 100%")
				}
				return nil
			}),
		),
		validation.Field(&dc.TenderEscalationOfferTTLMinutes,
			validation.By(func(_ any) error {
				if !validTenderEscalationTTL(dc.TenderEscalationOfferTTLMinutes) {
					return errors.New("Escalation offer expiry must be between 5 and 10080 minutes")
				}
				return nil
			}),
		),
		validation.Field(&dc.TenderMaxEscalations,
			validation.Min(int16(0)).Error("Escalation rounds must be between 0 and 5"),
			validation.Max(MaxTenderEscalations).Error("Escalation rounds must be between 0 and 5"),
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

func (dc *DispatchControl) BeforeCreate(ctx context.Context) error {
	if dc.AutoAssignmentStrategy == "" {
		dc.AutoAssignmentStrategy = AutoAssignmentStrategyProximity
	}
	if dc.EtaRiskThresholdMinutes == 0 {
		dc.EtaRiskThresholdMinutes = 60
	}
	if dc.EtaDwellMinutes == 0 {
		dc.EtaDwellMinutes = 60
	}
	if dc.ExceptionMaxEscalations == 0 {
		dc.ExceptionMaxEscalations = 3
	}
	if dc.ExceptionMissedStopGraceMinutes == 0 {
		dc.ExceptionMissedStopGraceMinutes = 30
	}
	if dc.ComplianceEnforcementLevel == "" {
		dc.ComplianceEnforcementLevel = ComplianceEnforcementLevelWarning
	}
	if dc.RecordServiceFailures == "" {
		dc.RecordServiceFailures = ServiceIncidentTypeNever
	}
	if dc.ServiceFailureGracePeriod == nil {
		defaultGracePeriod := DefaultServiceFailureGracePeriod
		dc.ServiceFailureGracePeriod = &defaultGracePeriod
	}
	dc.NormalizeAutoAssignmentSettings()
	dc.NormalizeExceptionSettings()
	dc.NormalizeTenderSettings()

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
		ExceptionDueInfoMinutes:              DefaultExceptionDueInfoMinutes,
		ExceptionDueWarningMinutes:           DefaultExceptionDueWarningMinutes,
		ExceptionDueCriticalMinutes:          DefaultExceptionDueCriticalMinutes,
		ExceptionEscalationMinutes:           DefaultExceptionEscalationMinutes,
		ExceptionMaxEscalations:              DefaultExceptionMaxEscalations,
		ExceptionMissedStopGraceMinutes:      DefaultExceptionMissedStopGraceMinutes,
		ExceptionPODGraceHours:               DefaultExceptionPODGraceHours,
		TenderCounterMarginFloorPercent: decimal.NewNullDecimal(
			DefaultTenderCounterMarginFloor,
		),
	}
}

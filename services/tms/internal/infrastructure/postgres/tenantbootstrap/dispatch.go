package tenantbootstrap

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const autoAssignConfidenceThreshold = "0.85"

func newDispatchControl(scope tenantscope.Scope) *dispatchcontrol.DispatchControl {
	tender := dispatchcontrol.DefaultTenderPolicy()
	grace := dispatchcontrol.DefaultServiceFailureGracePeriod

	return &dispatchcontrol.DispatchControl{
		ID:                              pulid.MustNew("dc_"),
		OrganizationID:                  scope.OrganizationID,
		BusinessUnitID:                  scope.BusinessUnitID,
		AutoAssignmentStrategy:          dispatchcontrol.AutoAssignmentStrategyProximity,
		AutoAssignConfidenceThreshold:   decimal.RequireFromString(autoAssignConfidenceThreshold),
		AutoAssignPlanningHorizonHours:  dispatchcontrol.DefaultAutoAssignPlanningHorizonHours,
		PlanningMode:                    dispatchcontrol.PlanningModeImmediate,
		HorizonMaxMovesPerDriver:        dispatchcontrol.DefaultHorizonMaxMovesPerDriver,
		CoverageRiskWindowHours:         dispatchcontrol.DefaultCoverageRiskWindowHours,
		CheckCallIntervalHours:          dispatchcontrol.DefaultCheckCallIntervalHours,
		EnableEtaAlerts:                 true,
		EtaRiskThresholdMinutes:         dispatchcontrol.DefaultEtaRiskThresholdMinutes,
		EtaDwellMinutes:                 dispatchcontrol.DefaultEtaDwellMinutes,
		ExceptionDueInfoMinutes:         dispatchcontrol.DefaultExceptionDueInfoMinutes,
		ExceptionDueWarningMinutes:      dispatchcontrol.DefaultExceptionDueWarningMinutes,
		ExceptionDueCriticalMinutes:     dispatchcontrol.DefaultExceptionDueCriticalMinutes,
		ExceptionEscalationMinutes:      dispatchcontrol.DefaultExceptionEscalationMinutes,
		ExceptionMaxEscalations:         dispatchcontrol.DefaultExceptionMaxEscalations,
		ExceptionMissedStopGraceMinutes: dispatchcontrol.DefaultExceptionMissedStopGraceMinutes,
		ExceptionPODGraceHours:          dispatchcontrol.DefaultExceptionPODGraceHours,
		TenderCounterOffersEnabled:      tender.CountersEnabled,
		TenderCounterMarginFloorPercent: decimal.NewNullDecimal(tender.CounterMarginFloor),
		TenderMaxEscalations:            tender.MaxEscalations,
		TenderCounterDecisionMinutes:    dispatchcontrol.DefaultTenderCounterDecisionMinutes,
		TenderExhaustedAction:           dispatchcontrol.TenderExhaustedActionNotify,
		TenderEscalationOfferTTLMinutes: dispatchcontrol.DefaultTenderEscalationTTLMinutes,
		ComplianceEnforcementLevel:      dispatchcontrol.ComplianceEnforcementLevelWarning,
		RecordServiceFailures:           dispatchcontrol.ServiceIncidentTypeNever,
		ServiceFailureGracePeriod:       &grace,
	}
}

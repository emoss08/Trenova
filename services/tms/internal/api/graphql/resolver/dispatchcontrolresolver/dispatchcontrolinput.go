package dispatchcontrolresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlinput"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
)

func dispatchControlFromInput(
	existing *dispatchcontrol.DispatchControl,
	input *gqlmodel.UpdateDispatchControlInput,
) (*dispatchcontrol.DispatchControl, error) {
	parse := gqlinput.NewParser()
	entity := &dispatchcontrol.DispatchControl{
		ID:                                   existing.ID,
		BusinessUnitID:                       existing.BusinessUnitID,
		OrganizationID:                       existing.OrganizationID,
		Version:                              int64(input.Version),
		CreatedAt:                            existing.CreatedAt,
		UpdatedAt:                            existing.UpdatedAt,
		EnableAutoAssignment:                 input.EnableAutoAssignment,
		AutoAssignmentStrategy:               input.AutoAssignmentStrategy,
		EnforceWorkerAssign:                  input.EnforceWorkerAssign,
		EnforceTrailerContinuity:             input.EnforceTrailerContinuity,
		EnforceHOSCompliance:                 input.EnforceHosCompliance,
		EnforceWorkerPTARestrictions:         input.EnforceWorkerPtaRestrictions,
		EnforceWorkerTractorFleetContinuity:  input.EnforceWorkerTractorFleetContinuity,
		EnforceDriverQualificationCompliance: input.EnforceDriverQualificationCompliance,
		EnforceMedicalCertCompliance:         input.EnforceMedicalCertCompliance,
		EnforceHazmatCompliance:              input.EnforceHazmatCompliance,
		EnforceDrugAndAlcoholCompliance:      input.EnforceDrugAndAlcoholCompliance,
		EnableAutoStopActuals:                input.EnableAutoStopActuals,
		ScoringWeights:                       scoringWeightsFromInput(parse, input.ScoringWeights),
		AutoAssignConfidenceThreshold: parse.Decimal(
			"autoAssignConfidenceThreshold",
			input.AutoAssignConfidenceThreshold,
		),
		AutoAssignMaxDeadheadMiles: parse.Int32Ptr(
			"autoAssignMaxDeadheadMiles",
			input.AutoAssignMaxDeadheadMiles,
		),
		AutoAssignPlanningHorizonHours: parse.Int16(
			"autoAssignPlanningHorizonHours",
			input.AutoAssignPlanningHorizonHours,
		),
		PlanningMode: input.PlanningMode,
		HorizonMaxMovesPerDriver: parse.Int16(
			"horizonMaxMovesPerDriver",
			input.HorizonMaxMovesPerDriver,
		),
		HorizonSearchIterations: parse.Int16Ptr(
			"horizonSearchIterations",
			input.HorizonSearchIterations,
		),
		CoverageRiskWindowHours: parse.Int16(
			"coverageRiskWindowHours",
			input.CoverageRiskWindowHours,
		),
		EnableCheckCallMonitoring: input.EnableCheckCallMonitoring,
		CheckCallIntervalHours: parse.Int16(
			"checkCallIntervalHours",
			input.CheckCallIntervalHours,
		),
		EnableEtaAlerts: input.EnableEtaAlerts,
		EtaRiskThresholdMinutes: parse.Int16(
			"etaRiskThresholdMinutes",
			input.EtaRiskThresholdMinutes,
		),
		EtaDwellMinutes: parse.Int16("etaDwellMinutes", input.EtaDwellMinutes),
		ExceptionDueInfoMinutes: parse.Int32(
			"exceptionDueInfoMinutes",
			input.ExceptionDueInfoMinutes,
		),
		ExceptionDueWarningMinutes: parse.Int32(
			"exceptionDueWarningMinutes",
			input.ExceptionDueWarningMinutes,
		),
		ExceptionDueCriticalMinutes: parse.Int32(
			"exceptionDueCriticalMinutes",
			input.ExceptionDueCriticalMinutes,
		),
		ExceptionEscalationMinutes: parse.Int32(
			"exceptionEscalationMinutes",
			input.ExceptionEscalationMinutes,
		),
		ExceptionMaxEscalations: parse.Int16(
			"exceptionMaxEscalations",
			input.ExceptionMaxEscalations,
		),
		ExceptionDefaultAssigneeID: parse.OptionalID(
			"exceptionDefaultAssigneeId",
			input.ExceptionDefaultAssigneeID,
		),
		ExceptionEscalationContactID: parse.OptionalID(
			"exceptionEscalationContactId",
			input.ExceptionEscalationContactID,
		),
		ExceptionMissedStopGraceMinutes: parse.Int16(
			"exceptionMissedStopGraceMinutes",
			input.ExceptionMissedStopGraceMinutes,
		),
		ExceptionPODGraceHours: parse.Int16(
			"exceptionPodGraceHours",
			input.ExceptionPodGraceHours,
		),
		TenderCounterOffersEnabled: input.TenderCounterOffersEnabled,
		TenderCounterMarginFloorPercent: parse.NullDecimal(
			"tenderCounterMarginFloorPercent",
			input.TenderCounterMarginFloorPercent,
		),
		TenderAutoAcceptCounters: input.TenderAutoAcceptCounters,
		TenderCounterDecisionMinutes: parse.Int32(
			"tenderCounterDecisionMinutes",
			input.TenderCounterDecisionMinutes,
		),
		TenderExhaustedAction: input.TenderExhaustedAction,
		TenderEscalationRateIncreasePercent: parse.Decimal(
			"tenderEscalationRateIncreasePercent",
			input.TenderEscalationRateIncreasePercent,
		),
		TenderEscalationOfferTTLMinutes: parse.Int32(
			"tenderEscalationOfferTtlMinutes",
			input.TenderEscalationOfferTTLMinutes,
		),
		TenderMaxEscalations: parse.Int16(
			"tenderMaxEscalations",
			input.TenderMaxEscalations,
		),
		ComplianceEnforcementLevel: input.ComplianceEnforcementLevel,
		RecordServiceFailures:      input.RecordServiceFailures,
		ServiceFailureTarget:       input.ServiceFailureTarget,
		ServiceFailureGracePeriod:  input.ServiceFailureGracePeriod,
	}
	if err := parse.Err(); err != nil {
		return nil, err
	}

	return entity, nil
}

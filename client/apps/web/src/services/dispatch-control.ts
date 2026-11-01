import {
  DispatchControlDocument,
  UpdateDispatchControlDocument,
  type UpdateDispatchControlInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { dispatchControlSchema, type DispatchControl } from "@/types/dispatch-control";

export function toDispatchControlInput(data: DispatchControl): UpdateDispatchControlInput {
  return {
    version: data.version ?? 0,
    enableAutoAssignment: data.enableAutoAssignment,
    autoAssignmentStrategy: data.autoAssignmentStrategy,
    enforceWorkerAssign: data.enforceWorkerAssign,
    enforceTrailerContinuity: data.enforceTrailerContinuity,
    enforceHosCompliance: data.enforceHosCompliance,
    enforceWorkerPtaRestrictions: data.enforceWorkerPtaRestrictions,
    enforceWorkerTractorFleetContinuity: data.enforceWorkerTractorFleetContinuity,
    enforceDriverQualificationCompliance: data.enforceDriverQualificationCompliance,
    enforceMedicalCertCompliance: data.enforceMedicalCertCompliance,
    enforceHazmatCompliance: data.enforceHazmatCompliance,
    enforceDrugAndAlcoholCompliance: data.enforceDrugAndAlcoholCompliance,
    enableAutoStopActuals: data.enableAutoStopActuals,
    scoringWeights: data.scoringWeights ?? {},
    autoAssignConfidenceThreshold: String(data.autoAssignConfidenceThreshold ?? 0),
    autoAssignMaxDeadheadMiles: data.autoAssignMaxDeadheadMiles ?? null,
    autoAssignPlanningHorizonHours: data.autoAssignPlanningHorizonHours ?? 48,
    planningMode: data.planningMode,
    horizonMaxMovesPerDriver: data.horizonMaxMovesPerDriver,
    horizonSearchIterations: data.horizonSearchIterations ?? null,
    coverageRiskWindowHours: data.coverageRiskWindowHours ?? 12,
    enableCheckCallMonitoring: data.enableCheckCallMonitoring,
    checkCallIntervalHours: data.checkCallIntervalHours,
    enableEtaAlerts: data.enableEtaAlerts,
    etaRiskThresholdMinutes: data.etaRiskThresholdMinutes,
    etaDwellMinutes: data.etaDwellMinutes,
    exceptionDueInfoMinutes: data.exceptionDueInfoMinutes,
    exceptionDueWarningMinutes: data.exceptionDueWarningMinutes,
    exceptionDueCriticalMinutes: data.exceptionDueCriticalMinutes,
    exceptionEscalationMinutes: data.exceptionEscalationMinutes,
    exceptionMaxEscalations: data.exceptionMaxEscalations,
    exceptionDefaultAssigneeId: data.exceptionDefaultAssigneeId || null,
    exceptionEscalationContactId: data.exceptionEscalationContactId || null,
    exceptionMissedStopGraceMinutes: data.exceptionMissedStopGraceMinutes,
    exceptionPodGraceHours: data.exceptionPodGraceHours,
    tenderCounterOffersEnabled: data.tenderCounterOffersEnabled,
    tenderCounterMarginFloorPercent:
      data.tenderCounterMarginFloorPercent == null
        ? null
        : String(data.tenderCounterMarginFloorPercent),
    tenderAutoAcceptCounters: data.tenderAutoAcceptCounters,
    tenderCounterDecisionMinutes: data.tenderCounterDecisionMinutes,
    tenderExhaustedAction: data.tenderExhaustedAction,
    tenderEscalationRateIncreasePercent: String(data.tenderEscalationRateIncreasePercent ?? 0),
    tenderEscalationOfferTtlMinutes: data.tenderEscalationOfferTtlMinutes,
    tenderMaxEscalations: data.tenderMaxEscalations,
    complianceEnforcementLevel: data.complianceEnforcementLevel,
    recordServiceFailures: data.recordServiceFailures,
    serviceFailureTarget: data.serviceFailureTarget ?? null,
    serviceFailureGracePeriod: data.serviceFailureGracePeriod ?? null,
  };
}

export class DispatchControlService {
  public async get(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: DispatchControlDocument,
      operationName: "DispatchControl",
      signal: options?.signal,
    });

    return safeParse(dispatchControlSchema, response.dispatchControl, "Dispatch Control");
  }

  public async update(data: DispatchControl) {
    const response = await requestGraphQL({
      document: UpdateDispatchControlDocument,
      operationName: "UpdateDispatchControl",
      variables: { input: toDispatchControlInput(data) },
    });

    return safeParse(dispatchControlSchema, response.updateDispatchControl, "Dispatch Control");
  }
}

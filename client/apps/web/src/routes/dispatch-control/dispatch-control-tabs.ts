import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import type { DispatchControl } from "@/types/dispatch-control";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type DispatchControlTab =
  | "assignment"
  | "costs"
  | "learning"
  | "monitoring"
  | "exceptions"
  | "tenders"
  | "compliance";

export type DispatchControlField = keyof DispatchControl & string;

export type DispatchControlTabSpec = SettingsTabSpec<DispatchControlField> & {
  value: DispatchControlTab;
};

const TAB_LABELS = defineLabels({
  assignment: "Assignment",
  costs: "Planner costs",
  learning: "Learning",
  monitoring: "Monitoring",
  exceptions: "Exceptions",
  tenders: "Carrier tenders",
  compliance: "Compliance",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<DispatchControlTab, readonly DispatchControlField[]> = {
  assignment: [
    "enableAutoAssignment",
    "autoProposeCoverage",
    "autoAssignConfidenceThreshold",
    "autoAssignMaxDeadheadMiles",
    "autoAssignPlanningHorizonHours",
    "coverageRiskWindowHours",
  ],
  costs: ["economics"],
  learning: ["learnDispatchStyle", "planForEndMarket", "anticipateFreight"],
  monitoring: [
    "recordServiceFailures",
    "serviceFailureGracePeriod",
    "serviceFailureTarget",
    "enableCheckCallMonitoring",
    "checkCallIntervalHours",
    "enableEtaAlerts",
    "etaRiskThresholdMinutes",
    "etaDwellMinutes",
  ],
  exceptions: [
    "exceptionDueCriticalMinutes",
    "exceptionDueWarningMinutes",
    "exceptionDueInfoMinutes",
    "exceptionEscalationMinutes",
    "exceptionMaxEscalations",
    "exceptionDefaultAssigneeId",
    "exceptionEscalationContactId",
    "exceptionMissedStopGraceMinutes",
    "exceptionPodGraceHours",
  ],
  tenders: [
    "tenderCounterOffersEnabled",
    "tenderCounterMarginFloorPercent",
    "tenderAutoAcceptCounters",
    "tenderCounterDecisionMinutes",
    "tenderExhaustedAction",
    "tenderEscalationRateIncreasePercent",
    "tenderEscalationOfferTtlMinutes",
    "tenderMaxEscalations",
  ],
  compliance: [
    "enforceHosCompliance",
    "enforceMedicalCertCompliance",
    "enforceDriverQualificationCompliance",
    "enforceHazmatCompliance",
    "enforceDrugAndAlcoholCompliance",
    "complianceEnforcementLevel",
    "enforceWorkerAssign",
    "enforceTrailerContinuity",
    "enforceWorkerPtaRestrictions",
    "enforceWorkerTractorFleetContinuity",
  ],
};

const FLEET_TABS: ReadonlySet<DispatchControlTab> = new Set([
  "assignment",
  "costs",
  "learning",
  "compliance",
]);

const TAB_ORDER: readonly DispatchControlTab[] = [
  "assignment",
  "costs",
  "learning",
  "monitoring",
  "exceptions",
  "tenders",
  "compliance",
];

/**
 * The tabs an organization sees, in order: planning, what it costs, what it has learned
 * and DOT compliance only for one that runs a fleet, carrier tenders only for a brokerage.
 * The form and its loading state both read this, so the outline has the form's tabs.
 */
export function dispatchControlTabs({
  runsAssets,
  brokers,
}: {
  runsAssets: boolean;
  brokers: boolean;
}): DispatchControlTabSpec[] {
  return TAB_ORDER.filter((tab) => {
    if (FLEET_TABS.has(tab)) return runsAssets;
    if (tab === "tenders") return brokers;
    return true;
  }).map((tab) => ({ value: tab, label: TAB_LABELS[tab], fields: TAB_FIELDS[tab] }));
}

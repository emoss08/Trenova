import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import type { DispatchControl } from "@/types/dispatch-control";
import { PlanFeature } from "@/lib/plan-feature";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import {
  hasOrganizationCapability,
  OrganizationCapability,
} from "@trenova/shared/types/organization-capability";
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

/** Which of the add-ons behind the tabs the organization's plan includes. */
export type DispatchControlPlan = {
  /** Dispatch Intelligence: the planner that assigns, what it costs and what it learns. */
  planner: boolean;
  /** Trenova AI's watchtower, which raises and escalates the exceptions. */
  watchtower: boolean;
  /** Carrier & Brokerage, which tenders loads to carriers. */
  brokerage: boolean;
};

/** What the organization's plan includes of the add-ons behind the tabs. */
export function useDispatchControlPlan(): DispatchControlPlan {
  const planner = usePlanFeature(PlanFeature.DispatchIntelligence).available;
  const watchtower = usePlanFeature(PlanFeature.AIWatchtower).available;
  const brokerage = usePlanFeature(PlanFeature.CarrierBrokerage).available;
  return { planner, watchtower, brokerage };
}

/** The tabs that belong to the planner, which is Dispatch Intelligence. */
const PLANNER_TABS: ReadonlySet<DispatchControlTab> = new Set(["assignment", "costs", "learning"]);

/**
 * The tabs an organization sees, in order: planning, what it costs, what it has learned
 * and DOT compliance only for one that runs a fleet, carrier tenders only for a brokerage.
 * A tab whose add-on the plan leaves out is not offered at all. The form and its loading
 * state both read this, so the outline has the form's tabs.
 */
export function dispatchControlTabs({
  runsAssets,
  brokers,
  plan,
}: {
  runsAssets: boolean;
  brokers: boolean;
  plan: DispatchControlPlan;
}): DispatchControlTabSpec[] {
  return TAB_ORDER.filter((tab) => {
    if (PLANNER_TABS.has(tab) && !plan.planner) return false;
    if (tab === "exceptions" && !plan.watchtower) return false;
    if (FLEET_TABS.has(tab)) return runsAssets;
    if (tab === "tenders") return brokers && plan.brokerage;
    return true;
  }).map((tab) => ({ value: tab, label: TAB_LABELS[tab], fields: TAB_FIELDS[tab] }));
}

/** The tabs this organization sees, for the form and its loading state alike. */
export function useDispatchControlTabs(): DispatchControlTabSpec[] {
  const capabilities = useOrgCapabilities();
  const plan = useDispatchControlPlan();

  return dispatchControlTabs({
    runsAssets: hasOrganizationCapability(capabilities, OrganizationCapability.AssetOperations),
    brokers: hasOrganizationCapability(capabilities, OrganizationCapability.Brokerage),
    plan,
  });
}

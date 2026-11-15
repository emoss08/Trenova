import { PlanFeature, type PlanFeatureKey } from "@/lib/plan-feature";
import { isFeatureOnPlan } from "@trenova/shared/hooks/use-plan-feature";
import type { UnavailableFeature } from "@trenova/shared/types/permission";

export type EmploymentView = "details" | "history";

export type ResolvedWorkerPanelTab = {
  tab: string;
  employmentView: EmploymentView;
};

/** The tab the panel opens on, and the one a tab the plan leaves out falls back to. */
export const WORKER_PANEL_HOME_TAB = "overview";

/**
 * The add-on feature each worker tab belongs to. A tab not listed is part of the base worker
 * record. A tab whose feature the plan leaves out is not drawn and its reads are not sent.
 */
const WORKER_TAB_FEATURES: Readonly<Record<string, PlanFeatureKey>> = {
  credentials: PlanFeature.WorkforceCompliance,
  testing: PlanFeature.WorkforceCompliance,
  dqf: PlanFeature.WorkforceCompliance,
  training: PlanFeature.WorkforceTalent,
  reviews: PlanFeature.WorkforceTalent,
  safety: PlanFeature.WorkforceSafety,
  pto: PlanFeature.WorkforceTimeOff,
  leave: PlanFeature.WorkforceTimeOff,
  schedule: PlanFeature.WorkforceTimeTracking,
  pay: PlanFeature.DriverPay,
  portal: PlanFeature.DriverPortal,
};

/** Whether the organization's plan includes the worker tab. */
export function isWorkerTabOnPlan(
  tab: string,
  unavailable: readonly UnavailableFeature[],
): boolean {
  return isFeatureOnPlan(unavailable, WORKER_TAB_FEATURES[tab]);
}

/**
 * Which tab the panel shows for a requested tab id. The employment history
 * used to be a tab of its own; links and concerns still say "timeline", and
 * that value lands on the Employment tab with the history view showing. A tab
 * the plan leaves out (a saved link, a concern) lands on the overview.
 */
export function resolveWorkerPanelTab(
  requested: string,
  employmentViewChoice: EmploymentView,
  unavailable: readonly UnavailableFeature[],
): ResolvedWorkerPanelTab {
  if (requested === "timeline") {
    return { tab: "employment", employmentView: "history" };
  }
  if (!isWorkerTabOnPlan(requested, unavailable)) {
    return { tab: WORKER_PANEL_HOME_TAB, employmentView: employmentViewChoice };
  }
  return { tab: requested, employmentView: employmentViewChoice };
}

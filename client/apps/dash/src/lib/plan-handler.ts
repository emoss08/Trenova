import {
  planRestrictionOfNotice,
  setPlanLimitHandler,
  type PlanLimitNotice,
} from "@trenova/shared/lib/plan-limit";
import { useSyncExternalStore } from "react";

/** The feature catalog key of the driver app; never shown. */
export const DRIVER_PORTAL_FEATURE = "tms.driver_portal";

let offPlan = false;
const listeners = new Set<() => void>();

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function snapshot(): boolean {
  return offPlan;
}

/** Whether the server has said the company's plan does not include the driver app. */
export function useDriverAppOffPlan(): boolean {
  return useSyncExternalStore(subscribe, snapshot, snapshot);
}

export function markDriverAppOffPlan(): void {
  if (offPlan) return;
  offPlan = true;
  for (const listener of listeners) listener();
}

/** For tests: back to not knowing. */
export function resetDriverAppPlan(): void {
  offPlan = false;
  for (const listener of listeners) listener();
}

/**
 * What the driver app does with a plan refusal from either transport. When the company's plan
 * does not include the driver app the API refuses everything it asks, sign-in included, so the
 * whole app gives way to one plain message instead of an error or a way back to sign-in that
 * leads to the same refusal. Any other refusal (an add-on a card needs) is left to the screen
 * that asked, whose error state names the add-on.
 */
export function handleDashPlanLimit(notice: PlanLimitNotice): boolean {
  const restriction = planRestrictionOfNotice(notice);
  if (restriction?.feature !== DRIVER_PORTAL_FEATURE) {
    return false;
  }
  markDriverAppOffPlan();
  return true;
}

export function installDashPlanLimitHandler(): void {
  setPlanLimitHandler(handleDashPlanLimit);
}

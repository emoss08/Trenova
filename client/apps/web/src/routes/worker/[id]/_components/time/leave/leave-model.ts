import type { LeaveCaseRow } from "@/lib/graphql/worker-leave";

export type EntitlementBar = {
  /** Share of the bar, 0 to 100, still left to take. */
  remainingPercent: number;
  /** Whole weeks the bar spans, one tick each. */
  weeks: number;
};

/**
 * The entitlement as a ruled bar: what is left as a share of the whole, never below zero or
 * past the end, ticked once a week.
 */
export function entitlementBar(entitlement: {
  remainingHours: string;
  totalHours: string;
  totalWeeks: string;
}): EntitlementBar {
  const remaining = Number(entitlement.remainingHours);
  const total = Number(entitlement.totalHours);
  const weeks = Math.max(1, Math.round(Number(entitlement.totalWeeks) || 0));
  if (!Number.isFinite(remaining) || !Number.isFinite(total) || total <= 0) {
    return { remainingPercent: 0, weeks };
  }
  return { remainingPercent: Math.min(100, Math.max(0, (remaining / total) * 100)), weeks };
}

/** Hours recorded against a case, counted or not. */
export function caseHours(leaveCase: Pick<LeaveCaseRow, "entries">): number {
  return leaveCase.entries.reduce((sum, entry) => sum + (Number(entry.hours) || 0), 0);
}

/** The FMLA eligibility test's hours half: 1,250 hours in the twelve months before leave. */
export const FMLA_ELIGIBILITY_HOURS = 1250;

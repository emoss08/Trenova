/**
 * Labels and shaping for the org chart: the departments headcount is read by,
 * and the scopes a manager can hand over while they are away.
 */

export const JOB_DEPARTMENT_ORDER = [
  "Operations",
  "Safety",
  "Maintenance",
  "Billing",
  "Administration",
  "Sales",
  "HumanResources",
  "Executive",
  "Other",
] as const;

export type JobDepartmentValue = (typeof JOB_DEPARTMENT_ORDER)[number];

export const JOB_DEPARTMENT_LABELS: Record<string, string> = {
  Operations: "Operations",
  Safety: "Safety",
  Maintenance: "Maintenance",
  Billing: "Billing",
  Administration: "Administration",
  Sales: "Sales",
  HumanResources: "Human Resources",
  Executive: "Executive",
  Other: "Other",
};

export function jobDepartmentLabel(value: string): string {
  return JOB_DEPARTMENT_LABELS[value] ?? value;
}

export const APPROVAL_SCOPE_LABELS: Record<string, string> = {
  All: "Everything",
  TimeOff: "Time off only",
  Expenses: "Expenses only",
};

export function approvalScopeLabel(value: string): string {
  return APPROVAL_SCOPE_LABELS[value] ?? value;
}

export const APPROVAL_SCOPE_HINTS: Record<string, string> = {
  All: "Every approval this manager can make.",
  TimeOff: "Time-off requests only. Expenses stay with the manager.",
  Expenses: "Expense claims only. Time off stays with the manager.",
};

export function approvalScopeHint(value: string): string {
  return APPROVAL_SCOPE_HINTS[value] ?? "";
}

export type DelegationState = "scheduled" | "active" | "ended" | "revoked";

/**
 * Where a delegation stands at an instant. Revoked wins over everything: a
 * delegation called back mid-window is not "active until Friday", it is over.
 */
export function delegationState(
  delegation: { startsAt: number; endsAt?: number | null; revokedAt?: number | null },
  now: number,
): DelegationState {
  if (delegation.revokedAt != null && delegation.revokedAt <= now) return "revoked";
  if (now < delegation.startsAt) return "scheduled";
  if (delegation.endsAt != null && now > delegation.endsAt) return "ended";
  return "active";
}

export const DELEGATION_STATE_LABELS: Record<DelegationState, string> = {
  scheduled: "Starts later",
  active: "In force",
  ended: "Finished",
  revoked: "Called back",
};

export function delegationStateTone(
  state: DelegationState,
): "active" | "inactive" | "warning" | "secondary" {
  switch (state) {
    case "active":
      return "active";
    case "revoked":
      return "inactive";
    case "scheduled":
      return "warning";
    default:
      return "secondary";
  }
}

/**
 * The share of a headcount one row carries, for a relative bar. Zero when
 * there is nobody to divide by: an empty roster has no distribution, not an
 * even one.
 */
export function headcountShare(workers: number, total: number): number {
  if (workers <= 0 || total <= 0) return 0;
  return Math.max(2, Math.min(100, Math.round((workers / total) * 100)));
}

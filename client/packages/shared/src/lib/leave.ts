/**
 * Labels and tones for leave of absence and FMLA. The regulation's own words
 * are used where it has them, so somebody reading this beside a WH-380 sees the
 * same phrases.
 */

export type LeaveTone = "active" | "inactive" | "warning" | "secondary" | "info";

export const MEASUREMENT_METHOD_LABELS: Record<string, string> = {
  CalendarYear: "Calendar year",
  HireAnniversary: "Twelve months from the hire anniversary",
  ForwardFromFirstUse: "Twelve months forward from first use",
  RollingBackward: "Rolling twelve months looking back",
};

export function measurementMethodLabel(value: string): string {
  return MEASUREMENT_METHOD_LABELS[value] ?? value;
}

export const MEASUREMENT_METHOD_HINTS: Record<string, string> = {
  CalendarYear:
    "Simple to explain, but an employee can take twelve weeks in December and twelve more in January.",
  HireAnniversary:
    "Each employee's own year. Same stacking risk as the calendar year, spread across the roster.",
  ForwardFromFirstUse: "The year starts the first day of leave taken, and runs twelve months on.",
  RollingBackward:
    "The only method that cannot be stacked into twenty-four weeks in a row, which is why most employers choose it.",
};

export function measurementMethodHint(value: string): string {
  return MEASUREMENT_METHOD_HINTS[value] ?? "";
}

export const LEAVE_CASE_STATUS_LABELS: Record<string, string> = {
  Pending: "Awaiting a decision",
  Approved: "Approved",
  Denied: "Denied",
  Closed: "Closed",
};

export function leaveCaseStatusLabel(value: string): string {
  return LEAVE_CASE_STATUS_LABELS[value] ?? value;
}

export function leaveCaseStatusTone(value: string): LeaveTone {
  switch (value) {
    case "Approved":
      return "active";
    case "Denied":
      return "inactive";
    case "Pending":
      return "warning";
    default:
      return "secondary";
  }
}

export const LEAVE_FREQUENCY_LABELS: Record<string, string> = {
  Continuous: "Continuous",
  Intermittent: "Intermittent",
  ReducedSchedule: "Reduced schedule",
};

export function leaveFrequencyLabel(value: string): string {
  return LEAVE_FREQUENCY_LABELS[value] ?? value;
}

export const CERTIFICATION_STATUS_LABELS: Record<string, string> = {
  NotRequired: "Not required",
  Requested: "Requested",
  Received: "Received",
  Insufficient: "Incomplete or insufficient",
  Overdue: "Overdue",
  Waived: "Waived",
};

export function certificationStatusLabel(value: string): string {
  return CERTIFICATION_STATUS_LABELS[value] ?? value;
}

export function certificationTone(value: string): LeaveTone {
  switch (value) {
    case "Received":
      return "active";
    case "Overdue":
    case "Insufficient":
      return "inactive";
    case "Requested":
      return "warning";
    default:
      return "secondary";
  }
}

/**
 * Whether the office is still waiting on paperwork the employee owes.
 */
export function certificationOutstanding(value: string): boolean {
  return value === "Requested" || value === "Insufficient" || value === "Overdue";
}

export const LEAVE_TYPE_LABELS: Record<string, string> = {
  FMLA: "FMLA",
  Medical: "Medical",
  Military: "Military",
  Parental: "Parental",
  Personal: "Personal",
  Other: "Other",
};

export function leaveTypeLabel(value: string): string {
  return LEAVE_TYPE_LABELS[value] ?? value;
}

/**
 * Renders an hours figure. Leave decimals cross the wire as strings so an
 * intermittent quarter-hour is not rounded away in transit.
 */
export function formatLeaveHours(value: string): string {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return value;
  return parsed.toLocaleString("en-US", { maximumFractionDigits: 2 });
}

/**
 * How much of the entitlement is gone, as a percentage for a progress bar.
 * Clamped to 100: designating leave after the fact can push usage past the
 * entitlement, and a bar past its own end reads as a rendering fault.
 */
export function entitlementUsedPercent(usedHours: string, totalHours: string): number {
  const used = Number(usedHours);
  const total = Number(totalHours);
  if (!Number.isFinite(used) || !Number.isFinite(total) || total <= 0) return 0;
  return Math.min(100, Math.round((used / total) * 100));
}

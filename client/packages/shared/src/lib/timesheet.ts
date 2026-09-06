/**
 * Time and attendance, shared by the clock, the approval queue and the payroll
 * file so the three never disagree about what a week adds up to.
 *
 * Everything here works in minutes, because that is what the server stores.
 * Hours are a rendering, and the one place they become a number rather than a
 * label is the payroll file, where two decimal places is what payroll systems
 * take.
 */

const MINUTES_PER_HOUR = 60;

/** Minutes as hours for a person to read. The part hour is kept: it is what overtime is usually made of. */
export function formatHours(minutes: number): string {
  if (!Number.isFinite(minutes) || minutes <= 0) return "0h";

  const whole = Math.floor(minutes / MINUTES_PER_HOUR);
  const rest = Math.round(minutes % MINUTES_PER_HOUR);
  if (whole === 0) return `${rest}m`;
  if (rest === 0) return `${whole}h`;
  return `${whole}h ${rest}m`;
}

/** Minutes as decimal hours, which is the unit a payroll file is read in. */
export function decimalHours(minutes: number): string {
  if (!Number.isFinite(minutes) || minutes <= 0) return "0.00";
  return (minutes / MINUTES_PER_HOUR).toFixed(2);
}

type TimesheetTotals = {
  regularMinutes: number;
  overtimeMinutes: number;
  paidLeaveMinutes: number;
};

export function totalHours(totals: TimesheetTotals): number {
  return totals.regularMinutes + totals.overtimeMinutes + totals.paidLeaveMinutes;
}

/**
 * How long somebody has been on the clock. It never goes negative: a clock read
 * a moment before the punch landed would otherwise show a shift running
 * backwards.
 */
export function elapsedMinutes(clockedInAt: number, nowSeconds: number): number {
  const seconds = nowSeconds - clockedInAt;
  if (seconds <= 0) return 0;
  return Math.floor(seconds / 60);
}

export type TimesheetAction = "submit" | "approve" | "reject";

/**
 * What the viewer may do with a week. A worker hands their own week over and
 * nothing else — offering them approve would send the server a request it is
 * going to refuse, and would suggest a sign-off that does not exist.
 */
export function timesheetActionsFor(
  status: string,
  viewer: { isOwner: boolean; canApprove: boolean },
): TimesheetAction[] {
  switch (status) {
    case "Open":
    case "Rejected":
      return ["submit"];
    case "Submitted":
      return viewer.canApprove && !viewer.isOwner ? ["approve", "reject"] : [];
    default:
      return [];
  }
}

export type TimesheetTone = {
  badge: string;
  label: string;
};

const TIMESHEET_TONES: Record<string, TimesheetTone> = {
  Open: {
    badge: "bg-muted text-muted-foreground border-transparent",
    label: "Open",
  },
  Submitted: {
    badge: "bg-amber-500/10 text-amber-700 dark:text-amber-300 border-amber-500/30",
    label: "Awaiting approval",
  },
  Approved: {
    badge: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/30",
    label: "Approved",
  },
  Rejected: {
    badge: "bg-rose-500/10 text-rose-700 dark:text-rose-300 border-rose-500/30",
    label: "Sent back",
  },
  Locked: {
    badge: "bg-blue-500/10 text-blue-700 dark:text-blue-300 border-blue-500/30",
    label: "Paid",
  },
};

export function timesheetStatusTone(status: string): TimesheetTone {
  return TIMESHEET_TONES[status] ?? TIMESHEET_TONES.Open;
}

export type PayrollCsvRow = {
  workerId: string;
  workerName: string;
  periodStart: number;
  periodEnd: number;
  regularMinutes: number;
  overtimeMinutes: number;
  paidLeaveMinutes: number;
};

const PAYROLL_HEADER = [
  "worker_id",
  "worker_name",
  "period_start",
  "period_end",
  "regular_hours",
  "overtime_hours",
  "paid_leave_hours",
];

/**
 * The payroll file. Dates go out as plain ISO days rather than epochs, because
 * the thing on the other end is a payroll system somebody imports into by hand.
 */
export function buildPayrollCsv(rows: readonly PayrollCsvRow[]): string {
  const lines = [PAYROLL_HEADER.join(",")];

  for (const row of rows) {
    lines.push(
      [
        csvValue(row.workerId),
        csvValue(row.workerName),
        csvValue(isoDay(row.periodStart)),
        csvValue(isoDay(row.periodEnd)),
        decimalHours(row.regularMinutes),
        decimalHours(row.overtimeMinutes),
        decimalHours(row.paidLeaveMinutes),
      ].join(","),
    );
  }

  return `${lines.join("\n")}\n`;
}

function isoDay(unixSeconds: number): string {
  return new Date(unixSeconds * 1000).toISOString().slice(0, 10);
}

// A name with a comma in it would otherwise split into two columns and shift
// everybody's hours one place to the right.
function csvValue(value: string): string {
  if (!/[",\n]/.test(value)) return value;
  return `"${value.replaceAll('"', '""')}"`;
}

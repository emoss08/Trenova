/**
 * Shift patterns and the rota, shared by the office board and the driver
 * portal so the two never disagree about what a pattern means.
 *
 * Everything here is indexed from Sunday, because the day mask stored on a
 * shift is. Two different week starts inside one feature is a bug waiting for
 * the first driver who works a Saturday.
 */

import { formatUnixInUserTimezone } from "./date";

const DAYS_IN_WEEK = 7;
const MINUTES_IN_DAY = 1440;
const SECONDS_IN_DAY = 86400;

export const DAY_LABELS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"] as const;

export const DAY_LABELS_LONG = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
] as const;

/** A mask is seven characters of 0 or 1; anything else is not a pattern. */
export function isDayMask(mask: string): boolean {
  return mask.length === DAYS_IN_WEEK && /^[01]{7}$/.test(mask);
}

/** The weekdays a mask covers, 0 for Sunday. */
export function dayMaskToDays(mask: string): number[] {
  if (!isDayMask(mask)) return [];

  const days: number[] = [];
  for (let index = 0; index < DAYS_IN_WEEK; index++) {
    if (mask[index] === "1") days.push(index);
  }
  return days;
}

export function daysToDayMask(days: readonly number[]): string {
  const mask = Array.from({ length: DAYS_IN_WEEK }, () => "0");
  for (const day of days) {
    if (Number.isInteger(day) && day >= 0 && day < DAYS_IN_WEEK) {
      mask[day] = "1";
    }
  }
  return mask.join("");
}

/**
 * A minute of the day as a clock time. It wraps, because a shift that ends at
 * 02:00 arrives here as minute 1560 rather than as a negative length.
 */
export function minutesToClock(minute: number): string {
  const wrapped = ((Math.trunc(minute) % MINUTES_IN_DAY) + MINUTES_IN_DAY) % MINUTES_IN_DAY;
  const hours = Math.floor(wrapped / 60);
  const minutes = wrapped % 60;
  return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}`;
}

/**
 * The inverse of minutesToClock: "06:30" to 390. Anything that is not a clock
 * time comes back as -1 so a form can refuse it rather than roster somebody
 * onto minute NaN.
 */
export function clockToMinutes(value: string): number {
  const match = /^(\d{1,2}):(\d{2})$/.exec(value.trim());
  if (!match) return -1;
  const hours = Number(match[1]);
  const minutes = Number(match[2]);
  if (hours > 23 || minutes > 59) return -1;
  return hours * MINUTES_PER_HOUR_LOCAL + minutes;
}

const MINUTES_PER_HOUR_LOCAL = 60;

function formatDuration(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (hours === 0) return `${rest}m`;
  if (rest === 0) return `${hours}h`;
  return `${hours}h ${rest}m`;
}

/**
 * When a shift runs and how long it is. A window that crosses midnight is
 * marked, because "20:00–06:00" on its own reads as a ten-hour gap rather than
 * a ten-hour shift.
 */
export function formatShiftWindow(startMinute: number, durationMinutes: number): string {
  const end = startMinute + durationMinutes;
  const crossesMidnight = end >= MINUTES_IN_DAY;
  const window = `${minutesToClock(startMinute)}–${minutesToClock(end)}${
    crossesMidnight ? " (+1)" : ""
  }`;
  return `${window} · ${formatDuration(durationMinutes)}`;
}

/**
 * The working days in words. Runs are collapsed — "Mon–Fri" rather than five
 * labels — because a rota is read at a glance.
 */
export function describeShiftPattern(mask: string, cycleWeeks: number): string {
  const days = dayMaskToDays(mask);
  if (days.length === 0) return "No working days";

  let description: string;
  if (days.length === DAYS_IN_WEEK) {
    description = "Every day";
  } else {
    const runs: string[] = [];
    let start = days[0];
    let previous = days[0];

    for (const day of days.slice(1)) {
      if (day === previous + 1) {
        previous = day;
        continue;
      }
      runs.push(formatRun(start, previous));
      start = day;
      previous = day;
    }
    runs.push(formatRun(start, previous));
    description = runs.join(", ");
  }

  if (cycleWeeks > 1) {
    return `${description} · ${cycleWeeks}-week rotation`;
  }
  return description;
}

/** Minutes a week the pattern works: the figure the office reasons about when it builds a shift. */
export function weeklyShiftMinutes(mask: string, durationMinutes: number): number {
  return dayMaskToDays(mask).length * Math.max(0, durationMinutes);
}

export const DAY_MASK_PRESETS = [
  { label: "Mon–Fri", mask: "0111110" },
  { label: "Mon–Sat", mask: "0111111" },
  { label: "Weekend", mask: "1000001" },
  { label: "Every day", mask: "1111111" },
] as const;

function formatRun(start: number, end: number): string {
  if (start === end) return DAY_LABELS[start];
  if (end === start + 1) return `${DAY_LABELS[start]}, ${DAY_LABELS[end]}`;
  return `${DAY_LABELS[start]}–${DAY_LABELS[end]}`;
}

/**
 * The Sunday midnight on or before an instant, which is the week the rota is
 * drawn for. It mirrors the server's `StartOfWeekUTC` exactly: the board is
 * keyed on UTC days, so a client that resolved the week in local time would
 * ask for a different week than the one it renders.
 */
export function startOfRotaWeek(atSeconds: number): number {
  const date = new Date(atSeconds * 1000);
  const midnight = Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()) / 1000;
  return midnight - date.getUTCDay() * SECONDS_IN_DAY;
}

/**
 * A rota day as a date. Always UTC: the board is keyed on UTC midnights, so
 * rendering a cell in the reader's own zone would label it with the wrong day
 * for anybody west of Greenwich.
 */
export function formatShiftDate(unixSeconds: number): string {
  return formatUnixInUserTimezone(unixSeconds, {
    month: "short",
    day: "numeric",
    year: "numeric",
    timezone: "UTC",
  });
}

export function addRotaWeeks(weekStart: number, weeks: number): number {
  return weekStart + weeks * DAYS_IN_WEEK * SECONDS_IN_DAY;
}

type RotaDayLike = {
  state: string;
  scheduled: boolean;
  durationMinutes: number;
  isConflict: boolean;
};

export type RotaRowSummary = {
  scheduledDays: number;
  conflicts: number;
  hours: number;
};

/**
 * What a worker's week adds up to. The hours count only the days they are
 * actually expected on: a day somebody is signed off is not an hour anybody is
 * working, and counting it would overstate the roster's cover.
 */
export function summariseRotaRow(row: { days: readonly RotaDayLike[] }): RotaRowSummary {
  let scheduledDays = 0;
  let conflicts = 0;
  let minutes = 0;

  for (const day of row.days) {
    if (day.isConflict) conflicts++;
    if (!day.scheduled) continue;
    scheduledDays++;
    if (!day.isConflict) minutes += day.durationMinutes;
  }

  return {
    scheduledDays,
    conflicts,
    hours: Math.round((minutes / 60) * 100) / 100,
  };
}

export type RotaTone = {
  cell: string;
  dot: string;
  label: string;
};

const ROTA_TONES: Record<string, RotaTone> = {
  Off: {
    cell: "bg-muted/30 text-muted-foreground border-transparent",
    dot: "bg-muted-foreground/40",
    label: "Off",
  },
  Scheduled: {
    cell: "bg-blue-500/10 text-blue-700 dark:text-blue-300 border-blue-500/30",
    dot: "bg-blue-500",
    label: "Scheduled",
  },
  Assigned: {
    cell: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/30",
    dot: "bg-emerald-500",
    label: "Assigned",
  },
  TimeOff: {
    cell: "bg-amber-500/10 text-amber-700 dark:text-amber-300 border-amber-500/30",
    dot: "bg-amber-500",
    label: "Time off",
  },
  Leave: {
    cell: "bg-purple-500/10 text-purple-700 dark:text-purple-300 border-purple-500/30",
    dot: "bg-purple-500",
    label: "Leave",
  },
  Unavailable: {
    cell: "bg-rose-500/10 text-rose-700 dark:text-rose-300 border-rose-500/30",
    dot: "bg-rose-500",
    label: "Unavailable",
  },
};

/** A state the board does not know how to colour still has to render. */
export function rotaStateTone(state: string): RotaTone {
  return ROTA_TONES[state] ?? ROTA_TONES.Off;
}

export const AVAILABILITY_TONES: Record<string, { badge: string; label: string }> = {
  Preferred: {
    badge: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/30",
    label: "Preferred",
  },
  Available: {
    badge: "bg-blue-500/10 text-blue-700 dark:text-blue-300 border-blue-500/30",
    label: "Available",
  },
  Unavailable: {
    badge: "bg-rose-500/10 text-rose-700 dark:text-rose-300 border-rose-500/30",
    label: "Unavailable",
  },
};

export const SWAP_STATUS_TONES: Record<string, { badge: string; label: string }> = {
  Proposed: {
    badge: "bg-blue-500/10 text-blue-700 dark:text-blue-300 border-blue-500/30",
    label: "Awaiting colleague",
  },
  Accepted: {
    badge: "bg-amber-500/10 text-amber-700 dark:text-amber-300 border-amber-500/30",
    label: "Awaiting approval",
  },
  Approved: {
    badge: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/30",
    label: "Approved",
  },
  Declined: {
    badge: "bg-rose-500/10 text-rose-700 dark:text-rose-300 border-rose-500/30",
    label: "Declined",
  },
  Rejected: {
    badge: "bg-rose-500/10 text-rose-700 dark:text-rose-300 border-rose-500/30",
    label: "Rejected",
  },
  Withdrawn: {
    badge: "bg-muted text-muted-foreground border-transparent",
    label: "Withdrawn",
  },
};

export type SwapAction = "accept" | "decline" | "withdraw";

/**
 * What the driver's own side of a swap lets them do. Only the driver a swap
 * was offered to can answer it, and only the one who proposed it can take it
 * back — offering both to both sides would send the server a request it is
 * going to refuse.
 *
 * Approving and rejecting are deliberately absent: a swap is decided by the
 * office.
 */
export function swapActionsFor(swap: { status: string; outgoing: boolean }): SwapAction[] {
  switch (swap.status) {
    case "Proposed":
      return swap.outgoing ? ["withdraw"] : ["accept", "decline"];
    case "Accepted":
      // Waiting on the office, and the proposer can still take it back until
      // the office answers.
      return swap.outgoing ? ["withdraw"] : [];
    default:
      return [];
  }
}

/** A swap still going somewhere, which is the queue a manager works. */
export function isSwapOpen(status: string): boolean {
  return status === "Proposed" || status === "Accepted";
}

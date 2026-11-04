import type {
  DetentionArrivalCell,
  DetentionFacility,
  DetentionIntelligence,
  DetentionIntelligenceTotals,
  DetentionRecoveryStep,
} from "@/lib/graphql/detention-intelligence";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type DetentionWindowValue = "30" | "90" | "180";

export const DETENTION_WINDOW_OPTIONS: {
  value: DetentionWindowValue;
  label: string;
  days: number;
}[] = [
  { value: "30", label: "30d", days: 30 },
  { value: "90", label: "90d", days: 90 },
  { value: "180", label: "180d", days: 180 },
];

const SECONDS_PER_DAY = 86_400;

/**
 * ATRI's 2023 driver survey: detention at 39.3% of stops. The one outside figure the page
 * shows, so a controller can tell whether their own share is high.
 */
export const INDUSTRY_DETAINED_SHARE = 0.393;

export function detentionWindowDays(value: DetentionWindowValue): number {
  return (
    DETENTION_WINDOW_OPTIONS.find((option) => option.value === value)?.days ??
    DETENTION_WINDOW_OPTIONS[1].days
  );
}

const DAYS_PER_WEEK = 7;
const dayFormatters = new Map<string, Intl.DateTimeFormat>();

/**
 * A moment's calendar day in a time zone, written as that day's midnight read as UTC: how the
 * server writes the Monday a week starts on, so the two can be compared.
 */
export function localDayStart(at: number, timezone: string): number {
  let formatter = dayFormatters.get(timezone);
  if (!formatter) {
    formatter = new Intl.DateTimeFormat("en-US", {
      timeZone: timezone,
      year: "numeric",
      month: "numeric",
      day: "numeric",
    });
    dayFormatters.set(timezone, formatter);
  }
  const parts = formatter.formatToParts(new Date(at * 1000));
  const part = (type: Intl.DateTimeFormatPartTypes) =>
    Number(parts.find((entry) => entry.type === type)?.value ?? 0);

  return Date.UTC(part("year"), part("month") - 1, part("day")) / 1000;
}

/**
 * How many of a week's seven days fall inside the window, counted in the window's time zone. The
 * window rarely starts on a Monday and the current week is never over, so the first and last
 * weeks usually hold fewer days and read low against the rest.
 */
export function weekDaysInWindow(
  weekStart: number,
  window: { from: number; to: number; timezone: string },
): number {
  const first = localDayStart(window.from, window.timezone);
  const last = localDayStart(window.to, window.timezone) + SECONDS_PER_DAY;
  const start = Math.max(weekStart, first);
  const end = Math.min(weekStart + DAYS_PER_WEEK * SECONDS_PER_DAY, last);

  return Math.max(0, Math.round((end - start) / SECONDS_PER_DAY));
}

export function hours(minutes: number): number {
  return Math.round(minutes / 60);
}

export function shareOf(part: number, whole: number): number {
  return whole > 0 ? part / whole : 0;
}

export function amount(value: string | null | undefined): number {
  return value === null || value === undefined ? 0 : Number(value);
}

export type CauseKey = "dock" | "open" | "early" | "late";

/**
 * Who caused the wait, in the order the server assigns each stop: a late arrival first, then an
 * open dock, then an early arrival, and only then the dock that was slow with the truck on time.
 * The accent is a category, not a severity.
 */
export const CAUSES: readonly { key: CauseKey; fillClass: string; bgClass: string }[] = [
  { key: "dock", fillClass: "fill-accent-indigo", bgClass: "bg-accent-indigo" },
  { key: "open", fillClass: "fill-accent-sky", bgClass: "bg-accent-sky" },
  { key: "early", fillClass: "fill-accent-teal", bgClass: "bg-accent-teal" },
  { key: "late", fillClass: "fill-accent-rose", bgClass: "bg-accent-rose" },
];

export const CAUSE_LABELS = defineLabels({
  dock: "On time, the dock was slow",
  open: "No appointment",
  early: "Arrived early",
  late: "Arrived late",
});

export const CAUSE_MEANING = defineLabels({
  dock: "The truck arrived inside its appointment and still waited past free time. This wait is the facility's.",
  open: "First come, first served docks, where the queue decides the wait.",
  early:
    "The driver arrived before the slot and waited for it. Most docks won't work a truck early.",
  late: "The truck missed its appointment, so the free time was forfeited.",
});

export const CAUSE_LEVER = defineLabels({
  dock: "Renegotiate free time, ask for drop trailers, or bring the scorecard to the next review.",
  open: "Ask the facility for appointments, or book the stop as a drop.",
  early: "Dispatch to the appointment, not ahead of it.",
  late: "Fix the plan: this time is ours to absorb.",
});

export function causeMinutes(
  row: Pick<
    DetentionIntelligenceTotals,
    "dockMinutes" | "openMinutes" | "earlyMinutes" | "lateMinutes"
  >,
  key: CauseKey,
): number {
  switch (key) {
    case "dock":
      return row.dockMinutes;
    case "open":
      return row.openMinutes;
    case "early":
      return row.earlyMinutes;
    case "late":
      return row.lateMinutes;
  }
}

/** The few facilities, most hours first, that reach `share` of every hour past free time. */
/**
 * "Fix these first" names a handful. Where detention is spread across many facilities the share
 * is never reached, and the group stops here rather than swallowing the list.
 */
export const MAX_FOCUS_FACILITIES = 10;

export function concentration(
  facilities: readonly DetentionFacility[],
  totalMinutes: number,
  share = 0.75,
): { count: number; share: number; minutes: number; ids: Set<string> } {
  const ids = new Set<string>();
  let minutes = 0;
  for (const facility of facilities) {
    minutes += facility.pastFreeMinutes;
    ids.add(facility.locationId);
    if (shareOf(minutes, totalMinutes) >= share || ids.size >= MAX_FOCUS_FACILITIES) {
      break;
    }
  }

  return { count: ids.size, share: shareOf(minutes, totalMinutes), minutes, ids };
}

export type FacilityRank = "hours" | "dock" | "detained" | "p90" | "billed" | "net";

/** What "Where to focus" can rank facilities by. Net margin is offered only with margins. */
export const FACILITY_RANKS: readonly FacilityRank[] = [
  "hours",
  "dock",
  "detained",
  "p90",
  "billed",
  "net",
];

export const FACILITY_RANK_LABELS = defineLabels({
  hours: "Hours past free time",
  dock: "On-time waiting",
  detained: "Share of stops detained",
  p90: "Slowest one in ten",
  billed: "Billed",
  net: "Net margin",
});

function rankScore(facility: DetentionFacility, rank: FacilityRank): number {
  switch (rank) {
    case "hours":
      return facility.pastFreeMinutes;
    case "dock":
      return facility.dockMinutes;
    case "detained":
      return shareOf(facility.detainedCount, facility.stopCount);
    case "p90":
      return facility.p90DwellMinutes;
    case "billed":
      return amount(facility.billedAmount);
    case "net":
      return amount(facility.netMargin);
  }
}

/**
 * The facilities in the order a rank puts them: the most first, except net margin, which puts
 * the biggest loss first. Ties keep the server's order, most hours first.
 */
export function rankFacilities(
  facilities: readonly DetentionFacility[],
  rank: FacilityRank,
): DetentionFacility[] {
  const direction = rank === "net" ? 1 : -1;
  return [...facilities].sort((a, b) => direction * (rankScore(a, rank) - rankScore(b, rank)));
}

/** Arrival hours folded into the bands dispatch books in. */
export const ARRIVAL_BANDS: readonly { key: ArrivalBandKey; from: number; to: number }[] = [
  { key: "night", from: 0, to: 6 },
  { key: "early", from: 6, to: 8 },
  { key: "morning", from: 8, to: 10 },
  { key: "late-morning", from: 10, to: 12 },
  { key: "midday", from: 12, to: 14 },
  { key: "afternoon", from: 14, to: 16 },
  { key: "late-afternoon", from: 16, to: 18 },
  { key: "evening", from: 18, to: 24 },
];

type ArrivalBandKey =
  | "night"
  | "early"
  | "morning"
  | "late-morning"
  | "midday"
  | "afternoon"
  | "late-afternoon"
  | "evening";

export const ARRIVAL_BAND_LABELS = defineLabels({
  night: "12–6a",
  early: "6–8a",
  morning: "8–10a",
  "late-morning": "10–12",
  midday: "12–2p",
  afternoon: "2–4p",
  "late-afternoon": "4–6p",
  evening: "6p–12",
});

export const WEEKDAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"] as const;

export const WEEKDAY_LABELS = defineLabels({
  Mon: "Mon",
  Tue: "Tue",
  Wed: "Wed",
  Thu: "Thu",
  Fri: "Fri",
  Sat: "Sat",
  Sun: "Sun",
});

/** Below this many arrivals a cell's share says nothing, so it is drawn as too few. */
export const MIN_ARRIVALS_PER_CELL = 5;

export type ArrivalBand = { stopCount: number; detainedCount: number };

/** Weekday by band, Monday first: [weekday][band]. */
export function foldArrivals(cells: readonly DetentionArrivalCell[]): ArrivalBand[][] {
  const grid = WEEKDAYS.map(() => ARRIVAL_BANDS.map(() => ({ stopCount: 0, detainedCount: 0 })));
  for (const cell of cells) {
    const day = cell.weekday - 1;
    const band = ARRIVAL_BANDS.findIndex((b) => cell.hour >= b.from && cell.hour < b.to);
    const slot = grid[day]?.[band];
    if (slot) {
      slot.stopCount += cell.stopCount;
      slot.detainedCount += cell.detainedCount;
    }
  }

  return grid;
}

export function arrivalPeak(
  grid: readonly ArrivalBand[][],
): { weekday: number; band: number; share: number } | null {
  let peak: { weekday: number; band: number; share: number } | null = null;
  grid.forEach((row, weekday) =>
    row.forEach((cell, band) => {
      if (cell.stopCount < MIN_ARRIVALS_PER_CELL) return;
      const share = shareOf(cell.detainedCount, cell.stopCount);
      if (!peak || share > peak.share) {
        peak = { weekday, band, share };
      }
    }),
  );

  return peak;
}

/** The share detained after noon, for the sentence that contrasts it with the peak. */
export function afternoonShare(grid: readonly ArrivalBand[][]): number {
  let stops = 0;
  let detained = 0;
  for (const row of grid) {
    for (const band of [4, 5, 6]) {
      stops += row[band]?.stopCount ?? 0;
      detained += row[band]?.detainedCount ?? 0;
    }
  }

  return shareOf(detained, stops);
}

export const RECOVERY_LABELS = defineLabels({
  Late: "Arrived late",
  Notice: "Notice missed its deadline",
  Waived: "Forgiven",
  Disputed: "Disputed",
  Pending: "Not yet billed",
  Other: "Not billable",
  Capped: "Caps and minimums",
});

/** The share of notices due that should go out by their deadline; below it, notice is a leak. */
export const NOTICE_ON_TIME_TARGET = 0.8;

export const DETENTION_POLICIES_PATH = "/detention/configuration-files/detention-policies";
export const DETENTION_DESK_PATH = "/detention/desk";

/**
 * What was billed less what drivers were paid for detention: the money the carrier kept. Null
 * for a reader who cannot see margins, since driver pay is withheld from them.
 */
export function keptAfterDriverPay(data: DetentionIntelligence): number | null {
  const driverPay = data.totals.driverPayAmount;
  return driverPay === null ? null : amount(data.recovery.billed) - amount(driverPay);
}

/** The money chain with zero-value steps left out, so every row the page draws carries money. */
export function recoverySteps(
  recovery: DetentionIntelligence["recovery"],
): DetentionRecoveryStep[] {
  return recovery.steps.filter((step) => amount(step.amount) > 0);
}

export function hasDetention(data: DetentionIntelligence | undefined): boolean {
  return (data?.totals.stopCount ?? 0) > 0;
}

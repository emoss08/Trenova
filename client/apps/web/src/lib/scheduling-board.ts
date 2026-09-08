import { weeklyShiftMinutes } from "@trenova/shared/lib/scheduling";

const SECONDS_IN_DAY = 86_400;

// The shapes below are structural rather than the generated rota types so the
// same maths serves the board, the overview and a fixture written from the
// schema by hand.
export type RotaDayLike = {
  date: number;
  state: string;
  scheduled: boolean;
  isConflict: boolean;
};

export type RotaRowLike = {
  workerId: string;
  name: string;
  fleetCode?: string | null;
  shiftCode?: string | null;
  shiftName?: string | null;
  scheduledDays: number;
  days: readonly RotaDayLike[];
};

export type DayCoverage = {
  date: number;
  /** People the pattern expects on the day, conflicts included. */
  expected: number;
  /** People actually available to work it: expected minus the conflicts. */
  covered: number;
  assigned: number;
  off: number;
  timeOff: number;
  leave: number;
  unavailable: number;
  conflicts: number;
};

function emptyCoverage(date: number): DayCoverage {
  return {
    date,
    expected: 0,
    covered: 0,
    assigned: 0,
    off: 0,
    timeOff: 0,
    leave: 0,
    unavailable: 0,
    conflicts: 0,
  };
}

/**
 * The board read down its columns instead of along its rows: how many people
 * each day actually has. A dispatcher plans a day, not a person, and the
 * per-row totals never answer "is Thursday thin".
 */
export function coverageByDay(rows: readonly Pick<RotaRowLike, "days">[]): DayCoverage[] {
  const columns = new Map<number, DayCoverage>();
  for (const row of rows) {
    for (const day of row.days) {
      let column = columns.get(day.date);
      if (!column) {
        column = emptyCoverage(day.date);
        columns.set(day.date, column);
      }
      if (day.scheduled) column.expected += 1;
      if (day.isConflict) column.conflicts += 1;
      else if (day.scheduled) column.covered += 1;
      switch (day.state) {
        case "Assigned":
          column.assigned += 1;
          break;
        case "Off":
          column.off += 1;
          break;
        case "TimeOff":
          column.timeOff += 1;
          break;
        case "Leave":
          column.leave += 1;
          break;
        case "Unavailable":
          column.unavailable += 1;
          break;
        default:
          break;
      }
    }
  }
  return Array.from(columns.values()).sort((a, b) => a.date - b.date);
}

export type CoverageTone = "strong" | "thin" | "none";

const THIN_COVERAGE_SHARE = 0.5;

/**
 * How a day reads against the busiest day on the board. The bar is relative
 * because a two-person yard and a two-hundred-driver terminal have no number
 * in common, but both know a day with half the usual cover is a thin one.
 */
export function coverageTone(covered: number, peak: number): CoverageTone {
  if (covered <= 0) return "none";
  if (peak > 0 && covered / peak < THIN_COVERAGE_SHARE) return "thin";
  return "strong";
}

export function peakCoverage(coverage: readonly DayCoverage[]): number {
  return coverage.reduce((peak, day) => Math.max(peak, day.covered), 0);
}

export function coverageOn(coverage: readonly DayCoverage[], instant: number): DayCoverage | null {
  return coverage.find((day) => day.date <= instant && instant < day.date + SECONDS_IN_DAY) ?? null;
}

export type RotaComposition = {
  /** Person-days the pattern has somebody working, whether or not dispatch has filled them. */
  working: number;
  off: number;
  /** Person-days lost to time off, leave and stated unavailability. */
  away: number;
  total: number;
};

export function rotaComposition(rows: readonly Pick<RotaRowLike, "days">[]): RotaComposition {
  const composition: RotaComposition = { working: 0, off: 0, away: 0, total: 0 };
  for (const row of rows) {
    for (const day of row.days) {
      composition.total += 1;
      switch (day.state) {
        case "Scheduled":
        case "Assigned":
          composition.working += 1;
          break;
        case "Off":
          composition.off += 1;
          break;
        default:
          composition.away += 1;
          break;
      }
    }
  }
  return composition;
}

export type RotaConflict = {
  workerId: string;
  name: string;
  date: number;
  /** What won over the pattern on the day: time off, leave or a stated unavailability. */
  state: string;
};

/** Every rostered day somebody cannot work, soonest first. */
export function rotaConflicts(
  rows: readonly Pick<RotaRowLike, "workerId" | "name" | "days">[],
): RotaConflict[] {
  const conflicts: RotaConflict[] = [];
  for (const row of rows) {
    for (const day of row.days) {
      if (!day.isConflict) continue;
      conflicts.push({ workerId: row.workerId, name: row.name, date: day.date, state: day.state });
    }
  }
  return conflicts.sort((a, b) => a.date - b.date || a.name.localeCompare(b.name));
}

export type UnrosteredWorker = {
  workerId: string;
  name: string;
  fleetCode: string | null;
  shiftName: string | null;
};

/**
 * People on the board with nothing to work. Somebody on a pattern that
 * rosters no days this week counts too: to the day's cover they are the same
 * gap as somebody on no pattern at all.
 */
export function unrosteredWorkers(
  rows: readonly Pick<
    RotaRowLike,
    "workerId" | "name" | "fleetCode" | "shiftName" | "scheduledDays"
  >[],
): UnrosteredWorker[] {
  return rows
    .filter((row) => row.scheduledDays === 0)
    .map(({ workerId, name, fleetCode, shiftName }) => ({
      workerId,
      name,
      fleetCode: fleetCode ?? null,
      shiftName: shiftName ?? null,
    }))
    .sort((a, b) => a.name.localeCompare(b.name));
}

/** Name, shift or terminal, matched loosely enough for a half-typed name. */
export function matchesRotaSearch(
  row: Pick<RotaRowLike, "name" | "fleetCode" | "shiftName" | "shiftCode">,
  query: string,
): boolean {
  const needle = query.trim().toLowerCase();
  if (!needle) return true;
  return [row.name, row.fleetCode, row.shiftName, row.shiftCode].some((field) =>
    (field ?? "").toLowerCase().includes(needle),
  );
}

export type SwapSummary = {
  /** Accepted by the colleague, waiting on the office. */
  awaitingOffice: number;
  /** Proposed and not yet answered by the colleague. */
  awaitingColleague: number;
};

export function summarizeSwaps(swaps: readonly { status: string }[]): SwapSummary {
  const summary: SwapSummary = { awaitingOffice: 0, awaitingColleague: 0 };
  for (const swap of swaps) {
    if (swap.status === "Accepted") summary.awaitingOffice += 1;
    else if (swap.status === "Proposed") summary.awaitingColleague += 1;
  }
  return summary;
}

export type TemplateStats = {
  active: number;
  retired: number;
  /** People on an active pattern, summed across the patterns. */
  onPatterns: number;
  /** The mean working week across active patterns, in minutes; null with none. */
  averageWeeklyMinutes: number | null;
};

export function templateStats(
  templates: readonly {
    status: string;
    activeAssignmentCount: number;
    daysOfWeek: string;
    durationMinutes: number;
  }[],
): TemplateStats {
  const stats: TemplateStats = { active: 0, retired: 0, onPatterns: 0, averageWeeklyMinutes: null };
  let minutes = 0;
  for (const template of templates) {
    if (template.status !== "Active") {
      stats.retired += 1;
      continue;
    }
    stats.active += 1;
    stats.onPatterns += template.activeAssignmentCount;
    minutes += weeklyShiftMinutes(template.daysOfWeek, template.durationMinutes);
  }
  stats.averageWeeklyMinutes = stats.active > 0 ? Math.round(minutes / stats.active) : null;
  return stats;
}

export function todayColumnIndex(days: readonly { date: number }[], today: number): number {
  return days.findIndex((day) => day.date <= today && today < day.date + SECONDS_IN_DAY);
}

export type RotaDensity = "comfortable" | "compact";

export const ROTA_DENSITIES: readonly RotaDensity[] = ["comfortable", "compact"];

export const ROTA_DENSITY_STORAGE_KEY = "scheduling.rota-density";

export function isRotaDensity(value: unknown): value is RotaDensity {
  return typeof value === "string" && (ROTA_DENSITIES as readonly string[]).includes(value);
}

/**
 * What a cell draws. `detail` is the start time and the length; `time` is the
 * start time alone; `block` is the tint and nothing else. More than one week
 * always draws blocks: fourteen or twenty-eight columns have no room for a
 * clock time, and the tooltip carries the window anyway.
 */
export type RotaCellMode = "detail" | "time" | "block";

export function rotaCellMode(density: RotaDensity, weeks: number): RotaCellMode {
  if (weeks > 1) return "block";
  return density === "compact" ? "time" : "detail";
}

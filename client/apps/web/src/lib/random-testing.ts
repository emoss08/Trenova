import { roundsPerYear } from "@trenova/shared/lib/drug-alcohol";

/**
 * Pure readings of the random testing programme: which rounds a year owes,
 * which of them have been drawn, and whether the draws are keeping pace with
 * the annual rates. Everything here takes the rows the GraphQL layer returns
 * and a clock, so it can be checked without a server.
 */

export type RandomPoolLike = {
  id: string;
  code: string;
  name: string;
  status: string;
  period: string;
  drugRatePercent: number;
  alcoholRatePercent: number;
  isDefault: boolean;
  meetsDotMinimums: boolean;
};

export type RandomDrawLike = {
  id: string;
  poolId: string;
  periodKey: string;
  periodStart: number;
  periodEnd: number;
  status: string;
  poolSize: number;
  drugTarget: number;
  alcoholTarget: number;
  drugSelected: number;
  alcoholSelected: number;
  drawnAt: number;
};

export type RandomEntryLike = {
  substance: string;
  status: string;
};

export const ROUND_STATUS_FILTERS = ["all", "Draft", "Final", "Cancelled"] as const;
export type RoundStatusFilter = (typeof ROUND_STATUS_FILTERS)[number];

export function isRoundStatusFilter(value: string): value is RoundStatusFilter {
  return (ROUND_STATUS_FILTERS as readonly string[]).includes(value);
}

export type RoundTone = "active" | "warning" | "inactive";

export function roundStatusTone(status: string): RoundTone {
  if (status === "Final") return "active";
  if (status === "Cancelled") return "inactive";
  return "warning";
}

export function roundStatusLabel(status: string): string {
  if (status === "Cancelled") return "Voided";
  return status;
}

/** A round that could not reach its targets because the pool was too small. */
export function drawShortOfTarget(draw: RandomDrawLike): boolean {
  return draw.drugSelected < draw.drugTarget || draw.alcoholSelected < draw.alcoholTarget;
}

function isLive(draw: RandomDrawLike): boolean {
  return draw.status !== "Cancelled";
}

const MONTH_LABELS = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
];

export type PeriodSlot = {
  /** The key the server stamps on a draw: 2026-Q1, 2026-M03, 2026-H1 or 2026. */
  key: string;
  label: string;
  start: number;
  end: number;
};

/**
 * The rounds one calendar year holds for a period, in the server's own key
 * format and UTC bounds, so a draw can be matched to its slot by key alone.
 */
export function periodSlotsForYear(period: string, year: number): PeriodSlot[] {
  const perYear = roundsPerYear(period);
  if (perYear === 0) return [];
  const months = 12 / perYear;
  const slots: PeriodSlot[] = [];
  for (let index = 0; index < perYear; index += 1) {
    const startMonth = index * months;
    const start = Date.UTC(year, startMonth, 1) / 1000;
    const end = Date.UTC(year, startMonth + months, 1) / 1000;
    slots.push({ key: slotKey(period, year, index), label: slotLabel(period, index), start, end });
  }
  return slots;
}

function slotKey(period: string, year: number, index: number): string {
  switch (period) {
    case "Monthly":
      return `${year}-M${String(index + 1).padStart(2, "0")}`;
    case "Quarterly":
      return `${year}-Q${index + 1}`;
    case "SemiAnnual":
      return `${year}-H${index + 1}`;
    default:
      return String(year);
  }
}

function slotLabel(period: string, index: number): string {
  switch (period) {
    case "Monthly":
      return MONTH_LABELS[index] ?? "";
    case "Quarterly":
      return `Q${index + 1}`;
    case "SemiAnnual":
      return `H${index + 1}`;
    default:
      return "Year";
  }
}

export function periodKeyFor(period: string, at: number): string {
  const date = new Date(at * 1000);
  const perYear = roundsPerYear(period);
  if (perYear === 0) return "";
  const months = 12 / perYear;
  return slotKey(period, date.getUTCFullYear(), Math.floor(date.getUTCMonth() / months));
}

export type SlotState = "final" | "draft" | "missed" | "due" | "upcoming";

export type CalendarSlot = PeriodSlot & {
  state: SlotState;
  draw: RandomDrawLike | null;
  /** Rounds drawn for this slot and then voided; they do not count. */
  voided: number;
};

export const SLOT_STATE_LABELS: Record<SlotState, string> = {
  final: "Final",
  draft: "Drawn, not final",
  missed: "Never drawn",
  due: "Owed now",
  upcoming: "Not yet",
};

/**
 * A pool's year as the auditor will read it: one slot per round the period
 * owes, each either drawn (draft or final), owed now, missed outright, or
 * still ahead. A voided round leaves its slot empty.
 */
export function poolCalendar(
  pool: RandomPoolLike,
  draws: readonly RandomDrawLike[],
  now: number,
  year = new Date(now * 1000).getUTCFullYear(),
): CalendarSlot[] {
  const own = draws.filter((draw) => draw.poolId === pool.id);
  return periodSlotsForYear(pool.period, year).map((slot) => {
    const matching = own.filter((draw) => draw.periodKey === slot.key);
    const live = matching.find(isLive) ?? null;
    const voided = matching.length - (live ? 1 : 0);
    let state: SlotState;
    if (live) {
      state = live.status === "Final" ? "final" : "draft";
    } else if (slot.end <= now) {
      state = "missed";
    } else if (slot.start <= now) {
      state = "due";
    } else {
      state = "upcoming";
    }
    return { ...slot, state, draw: live, voided };
  });
}

export type PoolProgress = {
  rounds: number;
  finalRounds: number;
  drugSelected: number;
  drugTarget: number;
  alcoholSelected: number;
  alcoholTarget: number;
  /** Every round drawn so far reached its targets. */
  onPace: boolean;
};

/**
 * How far a pool has got through the year's collections. Targets are summed
 * from the rounds actually drawn, because the target of a round not yet drawn
 * depends on a roster that does not exist yet.
 */
export function poolProgress(
  pool: RandomPoolLike,
  draws: readonly RandomDrawLike[],
  now: number,
  year = new Date(now * 1000).getUTCFullYear(),
): PoolProgress {
  const slots = periodSlotsForYear(pool.period, year);
  const yearStart = slots[0]?.start ?? 0;
  const yearEnd = slots[slots.length - 1]?.end ?? 0;
  const progress: PoolProgress = {
    rounds: 0,
    finalRounds: 0,
    drugSelected: 0,
    drugTarget: 0,
    alcoholSelected: 0,
    alcoholTarget: 0,
    onPace: true,
  };
  for (const draw of draws) {
    if (draw.poolId !== pool.id || !isLive(draw)) continue;
    if (draw.periodStart < yearStart || draw.periodStart >= yearEnd) continue;
    progress.rounds += 1;
    if (draw.status === "Final") progress.finalRounds += 1;
    progress.drugSelected += draw.drugSelected;
    progress.drugTarget += draw.drugTarget;
    progress.alcoholSelected += draw.alcoholSelected;
    progress.alcoholTarget += draw.alcoholTarget;
    if (drawShortOfTarget(draw)) progress.onPace = false;
  }
  return progress;
}

export type DrawFilter = {
  poolId: string | null;
  status: RoundStatusFilter;
};

/** Newest draw first, narrowed to a pool and a status when asked. */
export function filterDraws<TDraw extends RandomDrawLike>(
  draws: readonly TDraw[],
  filter: DrawFilter,
): TDraw[] {
  return draws
    .filter((draw) => (filter.poolId ? draw.poolId === filter.poolId : true))
    .filter((draw) => (filter.status === "all" ? true : draw.status === filter.status))
    .sort((left, right) => right.drawnAt - left.drawnAt);
}

export type ProgrammeOverview = {
  pools: number;
  activePools: number;
  belowMinimum: number;
  /** Drivers in the hat at the most recent live draw, or null before any draw. */
  lastPoolSize: number | null;
  lastDrawnAt: number | null;
  roundsThisYear: number;
  draftRounds: number;
  finalRounds: number;
  /** Active pools whose current round has not been drawn. */
  owedNow: number;
  /** Active pools with a past round of this year that was never drawn. */
  missed: number;
  drugSelected: number;
  drugTarget: number;
  alcoholSelected: number;
  alcoholTarget: number;
};

export function programmeOverview(
  pools: readonly RandomPoolLike[],
  draws: readonly RandomDrawLike[],
  now: number,
): ProgrammeOverview {
  const overview: ProgrammeOverview = {
    pools: pools.length,
    activePools: 0,
    belowMinimum: 0,
    lastPoolSize: null,
    lastDrawnAt: null,
    roundsThisYear: 0,
    draftRounds: 0,
    finalRounds: 0,
    owedNow: 0,
    missed: 0,
    drugSelected: 0,
    drugTarget: 0,
    alcoholSelected: 0,
    alcoholTarget: 0,
  };
  for (const pool of pools) {
    if (pool.status !== "Active") continue;
    overview.activePools += 1;
    if (!pool.meetsDotMinimums) overview.belowMinimum += 1;
    const calendar = poolCalendar(pool, draws, now);
    if (calendar.some((slot) => slot.state === "due")) overview.owedNow += 1;
    if (calendar.some((slot) => slot.state === "missed")) overview.missed += 1;
  }
  for (const pool of pools) {
    const progress = poolProgress(pool, draws, now);
    overview.roundsThisYear += progress.rounds;
    overview.finalRounds += progress.finalRounds;
    overview.draftRounds += progress.rounds - progress.finalRounds;
    overview.drugSelected += progress.drugSelected;
    overview.drugTarget += progress.drugTarget;
    overview.alcoholSelected += progress.alcoholSelected;
    overview.alcoholTarget += progress.alcoholTarget;
  }
  const latest = draws
    .filter(isLive)
    .reduce<RandomDrawLike | null>(
      (best, draw) => (best === null || draw.drawnAt > best.drawnAt ? draw : best),
      null,
    );
  if (latest) {
    overview.lastPoolSize = latest.poolSize;
    overview.lastDrawnAt = latest.drawnAt;
  }
  return overview;
}

export type EntryTally = {
  substance: "Drug" | "Alcohol";
  total: number;
  collected: number;
  notified: number;
  excused: number;
  missed: number;
  /** Selected or notified: the collection is still to come. */
  outstanding: number;
};

/** The selections of a round counted by substance, for the round's header. */
export function entryTally(entries: readonly RandomEntryLike[]): EntryTally[] {
  const tallies: EntryTally[] = (["Drug", "Alcohol"] as const).map((substance) => ({
    substance,
    total: 0,
    collected: 0,
    notified: 0,
    excused: 0,
    missed: 0,
    outstanding: 0,
  }));
  for (const entry of entries) {
    const tally = entry.substance === "Alcohol" ? tallies[1] : tallies[0];
    tally.total += 1;
    switch (entry.status) {
      case "Completed":
        tally.collected += 1;
        break;
      case "Notified":
        tally.notified += 1;
        tally.outstanding += 1;
        break;
      case "Excused":
        tally.excused += 1;
        break;
      case "Missed":
        tally.missed += 1;
        break;
      default:
        tally.outstanding += 1;
    }
  }
  return tallies;
}

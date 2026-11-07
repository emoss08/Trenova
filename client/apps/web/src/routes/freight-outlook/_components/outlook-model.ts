import type {
  FreightArea,
  FreightMove,
  FreightNetworkSlot,
  FreightOutlook,
  FreightPace,
  FreightSlot,
} from "@/lib/graphql/freight-outlook";

export type Horizon = "12" | "24" | "48";

export const HORIZON_OPTIONS: readonly { value: Horizon; slots: number }[] = [
  { value: "12", slots: 3 },
  { value: "24", slots: 6 },
  { value: "48", slots: 12 },
];

export const DEFAULT_HORIZON: Horizon = "24";
export const SLOT_SECONDS = 4 * 3600;
/** Below half a load (or half a driver) an area is called balanced, the server's threshold too. */
export const MIN_GAP = 0.5;
/** Short by a load and a half or more puts an area in the "Short of trucks" lane. */
export const SHORT_LANE = 1.5;
/** The customer pace thresholds, the same the server uses to suggest a call. */
const CALL_MIN_USUAL = 1;
const CALL_BEHIND_BY = 1;
const AHEAD_BY = 1.5;

export function horizonSlots(horizon: Horizon): number {
  return HORIZON_OPTIONS.find((option) => option.value === horizon)?.slots ?? 6;
}

/**
 * The slot indexes a reading covers: every window up to the horizon, or the one window a
 * dispatcher pinned on the tide.
 */
export function viewIndexes(slots: number, pinned: number | null): number[] {
  if (pinned !== null) {
    return [pinned];
  }
  return Array.from({ length: slots }, (_, index) => index);
}

export type BalanceKind = "short" | "spare" | "even";

export type BalanceLevel = { kind: BalanceKind; level: 0 | 1 | 2 | 3; amount: number };

/** One window of one area as a signed balance: short of trucks, trucks to spare, or even. */
export function balanceOf(slot: Pick<FreightSlot, "short" | "spare">): BalanceLevel {
  const kind: BalanceKind =
    slot.short >= MIN_GAP ? "short" : slot.spare >= MIN_GAP ? "spare" : "even";
  const amount = kind === "short" ? slot.short : kind === "spare" ? slot.spare : 0;
  const level = kind === "even" ? 0 : amount < 1.5 ? 1 : amount < 2.5 ? 2 : 3;
  return { kind, level, amount };
}

/** What a balance cell reads: "−3" for three trucks short, "+2" for two to spare, blank when even. */
export function balanceFigure(balance: BalanceLevel): string {
  if (balance.kind === "even") {
    return "";
  }
  const whole = Math.round(balance.amount);
  return balance.kind === "short" ? `−${whole}` : `+${whole}`;
}

export type Peak = { kind: BalanceKind; amount: number; index: number };

/** An area's worst window in a reading: its biggest shortfall, else its most spare. */
export function peakOf(area: FreightArea, indexes: readonly number[]): Peak {
  let short: Peak = { kind: "even", amount: 0, index: indexes[0] ?? 0 };
  let spare: Peak = { kind: "even", amount: 0, index: indexes[0] ?? 0 };
  for (const index of indexes) {
    const slot = area.slots[index];
    if (!slot) continue;
    if (slot.short > short.amount) short = { kind: "short", amount: slot.short, index };
    if (slot.spare > spare.amount) spare = { kind: "spare", amount: slot.spare, index };
  }
  if (short.amount >= MIN_GAP) return short;
  if (spare.amount >= MIN_GAP) return spare;
  return { kind: "even", amount: 0, index: indexes[0] ?? 0 };
}

export type Lane = "short" | "tight" | "spare" | "even";

export function laneOf(peak: Peak): Lane {
  if (peak.kind === "short") return peak.amount >= SHORT_LANE ? "short" : "tight";
  return peak.kind;
}

export type Totals = {
  uncovered: number;
  idle: number;
  shortAreas: number;
  spareAreas: number;
  worthCents: number;
  moves: number;
  networkShort: number;
  networkShortIndex: number;
};

/**
 * The figures a reading puts at the top: loads with no truck nearby (summed over every window),
 * drivers with nothing nearby (each area's worst window), the network's own gap, and what the
 * moves in the reading are worth.
 */
export function totalsOf(
  outlook: Pick<FreightOutlook, "areas" | "network" | "moves" | "slotStarts">,
  indexes: readonly number[],
): Totals {
  let uncovered = 0;
  let idle = 0;
  let shortAreas = 0;
  let spareAreas = 0;
  for (const area of outlook.areas) {
    let isShort = false;
    let worstSpare = 0;
    for (const index of indexes) {
      const slot = area.slots[index];
      if (!slot) continue;
      if (slot.short >= MIN_GAP) {
        uncovered += slot.short;
        isShort = true;
      }
      if (slot.spare >= MIN_GAP) worstSpare = Math.max(worstSpare, slot.spare);
    }
    if (isShort) shortAreas++;
    if (worstSpare > 0) {
      spareAreas++;
      idle += Math.round(worstSpare);
    }
  }

  let networkShort = 0;
  let networkShortIndex = indexes[0] ?? 0;
  for (const index of indexes) {
    const slot: FreightNetworkSlot | undefined = outlook.network[index];
    if (slot && slot.short > networkShort) {
      networkShort = slot.short;
      networkShortIndex = index;
    }
  }

  const moves = movesIn(outlook.moves, outlook.slotStarts, indexes);
  return {
    uncovered: Math.round(uncovered),
    idle,
    shortAreas,
    spareAreas,
    worthCents: moves.reduce((sum, move) => sum + (move.kind === "Call" ? 0 : move.netCents), 0),
    moves: moves.length,
    networkShort,
    networkShortIndex,
  };
}

/**
 * The moves for a reading: those whose window falls in it. A pinned window also keeps the moves
 * for the windows either side, since a driver sent for 4 AM matters at midnight too.
 */
export function movesIn(
  moves: readonly FreightMove[],
  slotStarts: readonly number[],
  indexes: readonly number[],
): FreightMove[] {
  if (slotStarts.length === 0 || indexes.length === 0) return [];
  const first = slotStarts[0];
  const reach = indexes.length === 1 ? 1 : 0;
  const low = Math.min(...indexes) - reach;
  const high = Math.max(...indexes) + reach;
  return moves.filter((move) => {
    const index = Math.floor((move.slotStart - first) / SLOT_SECONDS);
    return index >= low && index <= high;
  });
}

export type PaceStanding = "Behind" | "Ahead" | "OnPace";

export type PaceReading = {
  pace: FreightPace;
  usual: number;
  byNow: number;
  tendered: number;
  standing: PaceStanding;
  busiestStart: number;
};

export function paceStanding(usual: number, byNow: number, tendered: number): PaceStanding {
  if (usual >= CALL_MIN_USUAL && tendered <= byNow - CALL_BEHIND_BY) return "Behind";
  if (tendered >= byNow + AHEAD_BY) return "Ahead";
  return "OnPace";
}

const PACE_ORDER: Record<PaceStanding, number> = { Behind: 0, Ahead: 1, OnPace: 2 };

/** Each customer's freight in a reading against its own booking curve, the ones behind first. */
export function paceReadings(
  paces: readonly FreightPace[],
  indexes: readonly number[],
): PaceReading[] {
  const readings: PaceReading[] = [];
  for (const pace of paces) {
    let usual = 0;
    let byNow = 0;
    let tendered = 0;
    let busiest = -1;
    let busiestStart = pace.slots[indexes[0] ?? 0]?.start ?? 0;
    for (const index of indexes) {
      const slot = pace.slots[index];
      if (!slot) continue;
      usual += slot.usual;
      byNow += slot.byNow;
      tendered += slot.tendered;
      if (slot.usual > busiest) {
        busiest = slot.usual;
        busiestStart = slot.start;
      }
    }
    if (usual < 0.5 && tendered === 0) continue;
    readings.push({
      pace,
      usual,
      byNow,
      tendered,
      standing: paceStanding(usual, byNow, tendered),
      busiestStart,
    });
  }
  return readings.sort(
    (a, b) => PACE_ORDER[a.standing] - PACE_ORDER[b.standing] || b.usual - a.usual,
  );
}

/** The index of a window's start among the outlook's windows, or null when it is not one. */
export function slotIndexOf(slotStarts: readonly number[], start: number): number | null {
  if (slotStarts.length === 0) return null;
  const index = Math.floor((start - slotStarts[0]) / SLOT_SECONDS);
  return index >= 0 && index < slotStarts.length ? index : null;
}

/** Whether a window begins a new day in the reader's time zone, for the axis and the strips. */
export function startsDay(starts: readonly number[], index: number, dayOf: (at: number) => string) {
  return index === 0 || dayOf(starts[index]) !== dayOf(starts[index - 1]);
}

/** A move's identity across reads, for the hover the brief panel shares with the board. */
export function moveKey(move: FreightMove): string {
  return `${move.kind}:${move.workerId ?? move.customerId ?? ""}:${move.toCell}:${move.slotStart}`;
}

export type MoveTrail = {
  toCell: number;
  toIndex: number | null;
  fromCell: number | null;
  fromIndex: number | null;
};

/**
 * Where a move lands on the board: the window it fills in its target area and, for a driver who
 * travels there, the area and window that driver frees up in. A keep starts from its partner;
 * a call stays where it is.
 */
export function moveTrail(move: FreightMove, slotStarts: readonly number[]): MoveTrail {
  const toIndex = slotIndexOf(slotStarts, move.slotStart);
  const first = slotStarts[0] ?? 0;
  const origin =
    move.kind === "Send" || move.kind === "Book"
      ? { cell: move.fromCell, freeAt: move.freeAt }
      : move.kind === "Keep"
        ? { cell: move.partnerCell, freeAt: move.partnerFreeAt }
        : null;
  if (!origin || origin.cell === 0 || origin.cell === move.toCell) {
    return { toCell: move.toCell, toIndex, fromCell: null, fromIndex: null };
  }
  return {
    toCell: move.toCell,
    toIndex,
    fromCell: origin.cell,
    fromIndex: slotIndexOf(slotStarts, Math.max(origin.freeAt ?? first, first)),
  };
}

/** How many of the record's days landed inside their forecast range. */
export function daysInRange(
  days: readonly { low: number; high: number; actual: number }[],
): number {
  return days.filter((day) => day.actual >= day.low && day.actual <= day.high).length;
}

export type AreaRowEntry = { area: FreightArea; peak: Peak };

/** The board's groups, in the order a dispatcher works them, each worst first. */
export function laneGroups(
  areas: readonly FreightArea[],
  indexes: readonly number[],
): { lane: Lane; rows: AreaRowEntry[] }[] {
  const rows = areas.map((area) => ({ area, peak: peakOf(area, indexes) }));
  const order: readonly Lane[] = ["short", "tight", "spare", "even"];
  return order
    .map((lane) => ({
      lane,
      rows: rows
        .filter(({ peak }) => laneOf(peak) === lane)
        .sort((a, b) => b.peak.amount - a.peak.amount),
    }))
    .filter(({ rows: laneRows }) => laneRows.length > 0);
}

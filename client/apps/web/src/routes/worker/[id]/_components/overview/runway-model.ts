import type {
  WorkerClearanceLane,
  WorkerClearanceSource,
} from "@trenova/graphql/generated/graphql";
import type { WorkerOverview } from "@/lib/graphql/worker-overview";
import { workerDestinationTarget } from "@/lib/worker-concerns";
import { workerFormTarget, type WorkerTarget } from "@/routes/worker/use-worker-state";
import { SECONDS_IN_DAY } from "@/lib/time-units";

export type WorkerClearance = NonNullable<WorkerOverview["clearance"]>;
export type WorkerClearanceItem = WorkerClearance["items"][number];

const DAY_MS = SECONDS_IN_DAY * 1000;

export const RUNWAY_RANGES = [6, 12, 24] as const;
export type RunwayRange = (typeof RUNWAY_RANGES)[number];
export const DEFAULT_RUNWAY_RANGE: RunwayRange = 12;

export const RUNWAY_LANES = [
  "License",
  "Medical",
  "ReviewsAndQueries",
  "Training",
] as const satisfies readonly WorkerClearanceLane[];

export type RunwayVariant = "lanes" | "single";

/** How far off a date is: over 90 days, 31 to 90, 30 or less, or already past. */
export type RunwayTone = "far" | "soon" | "near" | "lapsed";

export function runwayTone(daysUntil: number): RunwayTone {
  if (daysUntil < 0) return "lapsed";
  if (daysUntil <= 30) return "near";
  if (daysUntil <= 90) return "soon";
  return "far";
}

function utcDayStart(unix: number): number {
  return Math.floor(unix / SECONDS_IN_DAY) * DAY_MS;
}

function daysInMonth(year: number, month: number): number {
  return new Date(Date.UTC(year, month + 1, 0)).getUTCDate();
}

/**
 * The days a range of months spans from the day of `asOf`, by the calendar (a 12-month range
 * from Nov 22, 2026 ends on Nov 22, 2027). A day the shorter month lacks ends on its last day.
 */
export function runwayRangeDays(asOf: number, months: RunwayRange): number {
  const start = new Date(utcDayStart(asOf));
  const year = start.getUTCFullYear();
  const month = start.getUTCMonth() + months;
  const day = Math.min(start.getUTCDate(), daysInMonth(year, month));
  return Math.round((Date.UTC(year, month, day) - start.getTime()) / DAY_MS);
}

/** Where a day count sits on the axis, from 0 (today) to 1 (the end of the range). */
export function runwayX(daysUntil: number, rangeDays: number): number {
  if (rangeDays <= 0) return 0;
  return Math.min(1, Math.max(0, daysUntil / rangeDays));
}

export type RunwayTick = {
  /** The first of the month, as unix seconds at UTC midnight. */
  at: number;
  x: number;
  /** The month opens a year other than the previous label's, so its label carries the year. */
  withYear: boolean;
};

/**
 * One tick on the first of each month inside the range, every third month past about 13
 * months so the labels never crowd. The first label in a new year carries its year.
 */
export function runwayTicks(asOf: number, rangeDays: number): RunwayTick[] {
  const startMs = utcDayStart(asOf);
  const start = new Date(startMs);
  const endMs = startMs + rangeDays * DAY_MS;
  const step = rangeDays > 400 ? 3 : 1;
  const ticks: RunwayTick[] = [];
  let year = start.getUTCFullYear();
  let cursor = Date.UTC(year, start.getUTCMonth() + 1, 1);
  for (let index = 0; cursor < endMs; index++) {
    const at = new Date(cursor);
    if (index % step === 0) {
      ticks.push({
        at: cursor / 1000,
        x: (cursor - startMs) / DAY_MS / rangeDays,
        withYear: at.getUTCFullYear() !== year,
      });
      year = at.getUTCFullYear();
    }
    cursor = Date.UTC(at.getUTCFullYear(), at.getUTCMonth() + 1, 1);
  }
  return ticks;
}

/** Whether an item keeps the worker off loads at its date (drawn solid). */
export function stopsAssignment(item: Pick<WorkerClearanceItem, "effect">): boolean {
  return item.effect === "StopsAssignment";
}

/** Whether an item ends the cleared band: it stops assignment, and not only hazmat loads. */
export function endsClearance(item: Pick<WorkerClearanceItem, "effect" | "hazmatOnly">): boolean {
  return stopsAssignment(item) && !item.hazmatOnly;
}

/** The earliest item that ends the cleared band, or null when none does. */
export function firstHardStop<
  T extends Pick<WorkerClearanceItem, "effect" | "hazmatOnly" | "daysUntil">,
>(items: readonly T[]): T | null {
  let first: T | null = null;
  for (const item of items) {
    if (!endsClearance(item)) continue;
    if (first === null || item.daysUntil < first.daysUntil) first = item;
  }
  return first;
}

export type RunwayStop = {
  key: string;
  name: string;
  source: WorkerClearanceSource;
  dueAt: number;
  daysUntil: number;
};

/**
 * The stop the cleared band runs to. It is read from every item, never from a lane's filtered
 * share, so the Credentials strip draws the same edge as the Overview; the server's first
 * stop covers one past the items it capped.
 */
export function clearanceStop(clearance: WorkerClearance): RunwayStop | null {
  return firstHardStop(clearance.items) ?? clearance.firstStop ?? null;
}

export type RunwayBand = {
  /** The band's right edge, from 0 to 1; 1 when the stop is past the range or there is none. */
  edge: number;
  stop: RunwayStop | null;
  /** The stop lies past the range, so the band runs off the right edge. */
  beyond: boolean;
  /** The edge sits so far right that its flag reads to the left of it. */
  flagFlipped: boolean;
};

/** The cleared band, or null when the worker is not cleared today. */
export function runwayBand(clearance: WorkerClearance, rangeDays: number): RunwayBand | null {
  if (!clearance.cleared) return null;
  const stop = clearanceStop(clearance);
  if (!stop) return { edge: 1, stop: null, beyond: true, flagFlipped: false };
  const edge = runwayX(stop.daysUntil, rangeDays);
  return { edge, stop, beyond: stop.daysUntil > rangeDays, flagFlipped: edge > 0.8 };
}

export type ClearanceSummary = {
  stop: RunwayStop | null;
  /** Items due from today until the stop, the stop itself left out. */
  dueBefore: number;
  /** Of those, the ones that stop hazmat loads only. */
  hazmatStopsBefore: number;
  /** Items due from today on, for a worker with no stop ahead. */
  dueAhead: number;
};

/** What the verdict sentence counts: what lapses first and how much comes due before it. */
export function clearanceSummary(clearance: WorkerClearance): ClearanceSummary {
  const stop = clearanceStop(clearance);
  const ahead = clearance.items.filter((item) => item.daysUntil >= 0);
  const before = stop
    ? ahead.filter((item) => item.key !== stop.key && item.daysUntil < stop.daysUntil)
    : [];
  return {
    stop,
    dueBefore: before.length,
    hazmatStopsBefore: before.filter((item) => stopsAssignment(item) && item.hazmatOnly).length,
    dueAhead: ahead.length,
  };
}

export function runwayLaneCount(variant: RunwayVariant): number {
  return variant === "single" ? 1 : RUNWAY_LANES.length;
}

export type RunwayMarker = {
  item: WorkerClearanceItem;
  lane: number;
  /** From 0 to 1; a lapsed item sits on the Today line, one past the range at the right edge. */
  x: number;
  inRange: boolean;
  tone: RunwayTone;
  solid: boolean;
  endsBand: boolean;
};

/** Every item as a marker in its lane, earliest first, limited to `sources` when given. */
export function runwayMarkers(
  clearance: WorkerClearance,
  options: {
    variant: RunwayVariant;
    rangeDays: number;
    sources?: readonly WorkerClearanceSource[];
  },
): RunwayMarker[] {
  const stop = clearanceStop(clearance);
  return clearance.items
    .filter((item) => !options.sources || options.sources.includes(item.source))
    .map((item) => ({
      item,
      lane: options.variant === "single" ? 0 : RUNWAY_LANES.indexOf(item.lane),
      x: runwayX(item.daysUntil, options.rangeDays),
      inRange: item.daysUntil <= options.rangeDays,
      tone: runwayTone(item.daysUntil),
      solid: stopsAssignment(item),
      endsBand: clearance.cleared && stop !== null && item.key === stop.key,
    }))
    .filter((marker) => marker.lane >= 0)
    .sort((a, b) => a.item.daysUntil - b.item.daysUntil || a.lane - b.lane);
}

/** The markers past the range, per lane, earliest first: the Later notch. */
export function runwayLater(markers: readonly RunwayMarker[], laneCount: number): RunwayMarker[][] {
  const lanes: RunwayMarker[][] = Array.from({ length: laneCount }, () => []);
  for (const marker of markers) {
    if (!marker.inRange) lanes[marker.lane]?.push(marker);
  }
  return lanes;
}

export type RunwayLabelSide = "above" | "below";

export type RunwayLabel = {
  side: RunwayLabelSide;
  /** The label reads to the left of its marker, because to its right it would leave the plot. */
  flip: boolean;
  /** Only the name and the day count, the date left to the tooltip. */
  short: boolean;
  /** No room for any label: the marker stands alone and its tooltip says everything. */
  hidden: boolean;
};

/** A label's width in pixels, in full or shortened. */
export type RunwayLabelMeasure = (item: WorkerClearanceItem, short: boolean) => number;

/** The space between a marker's centre and its label. */
export const RUNWAY_LABEL_GAP = 10;
/** The least space between two labels on one row. */
export const RUNWAY_LABEL_SPACING = 8;

/**
 * Lays each lane's labels out from left to right: above the track, else below it, else
 * shortened on whichever row has room, else none. A label that would run past the plot's
 * right edge reads to the left of its marker.
 */
export function placeRunwayLabels(
  markers: readonly RunwayMarker[],
  options: { width: number; measure: RunwayLabelMeasure; sides: readonly RunwayLabelSide[] },
): Map<string, RunwayLabel> {
  const { width, measure, sides } = options;
  const placed = new Map<string, RunwayLabel>();
  const lanes = new Map<number, RunwayMarker[]>();
  for (const marker of markers) {
    if (!marker.inRange) continue;
    const lane = lanes.get(marker.lane) ?? [];
    lane.push(marker);
    lanes.set(marker.lane, lane);
  }

  const span = (px: number, labelWidth: number) => {
    const start = px + RUNWAY_LABEL_GAP;
    if (start + labelWidth <= width) return { start, end: start + labelWidth, flip: false };
    const flipped = px - RUNWAY_LABEL_GAP - labelWidth;
    return { start: flipped, end: flipped + labelWidth, flip: true };
  };

  for (const lane of lanes.values()) {
    lane.sort((a, b) => a.x - b.x || a.item.daysUntil - b.item.daysUntil);
    const ends: Record<RunwayLabelSide, number> = {
      above: Number.NEGATIVE_INFINITY,
      below: Number.NEGATIVE_INFINITY,
    };
    for (const marker of lane) {
      const px = marker.x * width;
      let label: RunwayLabel | null = null;
      for (const short of [false, true]) {
        const at = span(px, measure(marker.item, short));
        const side = sides.find((candidate) => at.start >= ends[candidate] + RUNWAY_LABEL_SPACING);
        if (side) {
          ends[side] = at.end;
          label = { side, flip: at.flip, short, hidden: false };
          break;
        }
      }
      placed.set(
        marker.item.key,
        label ?? { side: sides[0] ?? "above", flip: false, short: true, hidden: true },
      );
    }
  }
  return placed;
}

export type ClearanceItemAction =
  | "renew"
  | "add-credential"
  | "log-query"
  | "open-training"
  | "open-review"
  | "start-review";

/**
 * What a marker's click does: a credential on file opens to be renewed and one never recorded
 * opens to be added, the Clearinghouse query to be logged, a course or a review where it is,
 * and a review not yet started opens to be started.
 */
export function clearanceItemAction(item: Pick<WorkerClearanceItem, "source" | "destination">): {
  action: ClearanceItemAction;
  target: WorkerTarget;
} {
  const { destination } = item;
  switch (item.source) {
    case "Credential":
      return destination.recordId
        ? {
            action: "renew",
            target: {
              ...workerFormTarget("renew-credential"),
              credential: destination.recordId,
            } as WorkerTarget,
          }
        : { action: "add-credential", target: workerFormTarget("add-credential") };
    case "ClearinghouseQuery":
      return { action: "log-query", target: workerFormTarget("log-query") };
    case "Training":
      return { action: "open-training", target: workerDestinationTarget(destination) };
    case "PerformanceReview":
      return destination.recordId
        ? { action: "open-review", target: workerDestinationTarget(destination) }
        : { action: "start-review", target: workerFormTarget("start-review") };
  }
}

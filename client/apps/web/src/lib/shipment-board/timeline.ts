import type { CapacityUnit } from "@/lib/graphql/shipment-board";
import { resolveCoverage, type Coverage } from "@/lib/shipment-board/coverage";
import { ACTIVE_STAGES, isActiveStage, type ActiveStage } from "@/lib/shipment-board/stage";
import { packLane, type PackedLane } from "@/lib/timeline/lanes";
import { dayBoundaries } from "@/lib/timeline/time-scale";
import { getStartOfDay } from "@trenova/shared/lib/date";
import type { Shipment, Stop } from "@trenova/shared/types/shipment";

export const TIMELINE_RANGES = ["day", "3d", "week"] as const;
export type TimelineRange = (typeof TIMELINE_RANGES)[number];

export const TIMELINE_ROW_MODES = ["shipments", "coverage"] as const;
export type TimelineRowMode = (typeof TIMELINE_ROW_MODES)[number];

export const RANGE_DAYS: Record<TimelineRange, number> = { day: 1, "3d": 3, week: 7 };

const HOUR = 3_600;
const DAY = 86_400;

/**
 * A driver's duty window under the 14-hour rule. A load picking up later than
 * this from now starts after a reset, so today's remaining drive time says
 * nothing about it.
 */
export const DUTY_WINDOW_SECONDS = 14 * HOUR;

/* ------------------------------------------------------------------ days */

/** "2026-11-03" for the day `unix` falls on in `timezone`. */
export function isoDay(unix: number, timezone: string): string {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(unix * 1000));
}

/** The start, in `timezone`, of the day an ISO date names; null for anything that is not one. */
export function dayStartOfIso(iso: string, timezone: string): number | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(iso)) return null;
  const noonUtc = Date.parse(`${iso}T12:00:00Z`);
  if (Number.isNaN(noonUtc)) return null;
  return getStartOfDay(new Date(noonUtc), timezone);
}

/** The first day a range shows: the anchor day, or the Monday of its week for a week. */
export function rangeFirstDay(
  range: TimelineRange,
  anchorDayStart: number,
  timezone: string,
): number {
  if (range !== "week") return anchorDayStart;
  const weekday = new Intl.DateTimeFormat("en-US", { timeZone: timezone, weekday: "short" }).format(
    new Date((anchorDayStart + 12 * HOUR) * 1000),
  );
  const fromMonday = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].indexOf(weekday);
  return dayBoundaries(anchorDayStart - Math.max(0, fromMonday) * DAY, 0, timezone)[0];
}

/* ------------------------------------------------------------------ loads */

export type TimelineStop = {
  id: string;
  type: Stop["type"];
  city: string;
  place: string;
  windowStart: number;
  windowEnd: number;
  arrivedAt: number | null;
  departedAt: number | null;
  done: boolean;
};

export type TimelineEta = {
  /** Index into the load's stops. */
  stopIndex: number;
  at: number;
  /** Minutes past the stop's window the projection lands; 0 when it makes it. */
  lateMinutes: number;
};

/** One shipment as the timeline draws it: its live stops on a time axis. */
export type TimelineLoad = {
  id: string;
  shipment: Shipment;
  stage: ActiveStage;
  stops: TimelineStop[];
  /** From the first window or arrival to the last window, departure or projected arrival. */
  start: number;
  end: number;
  eta: TimelineEta | null;
  /**
   * How much of the bar is driven: all of a delivered load, up to now for one
   * under way, none of one that has not left. Kept apart from the clock so a
   * minute passing redraws only the loads under way.
   */
  driven: "all" | "underway" | "none";
  coverage: Coverage;
  /** The first live move: the one an assignment goes on. */
  moveId: string | null;
  /** Driving minutes the load still needs, from its ETA projection; null when there is none. */
  driveMinutes: number | null;
};

function placeOf(stop: Stop): { city: string; place: string } {
  const city = stop.location?.city ?? stop.location?.name ?? "";
  const state = stop.location?.state?.abbreviation;
  return { city, place: [city, state].filter(Boolean).join(", ") };
}

function liveStops(shipment: Shipment): Stop[] {
  const out: Stop[] = [];
  for (const move of shipment.moves ?? []) {
    if (move.status === "Canceled") continue;
    for (const stop of move.stops ?? []) {
      if (stop.status !== "Canceled" && stop.scheduledWindowStart > 0) out.push(stop);
    }
  }
  return out;
}

/**
 * The loads the timeline can place: those with a stage it draws and at least
 * one live stop with an appointment.
 */
export function buildTimelineLoads(shipments: readonly Shipment[]): TimelineLoad[] {
  const loads: TimelineLoad[] = [];
  for (const shipment of shipments) {
    if (!shipment.id) continue;
    const stage = shipment.stage ?? "NeedsCoverage";
    if (!isActiveStage(stage)) continue;
    const raw = liveStops(shipment);
    if (raw.length === 0) continue;

    const stops: TimelineStop[] = raw.map((stop, index) => ({
      id: stop.id ?? `${shipment.id}-${index}`,
      type: stop.type,
      ...placeOf(stop),
      windowStart: stop.scheduledWindowStart,
      windowEnd: Math.max(
        stop.scheduledWindowEnd ?? stop.scheduledWindowStart,
        stop.scheduledWindowStart,
      ),
      arrivedAt: stop.actualArrival ?? null,
      departedAt: stop.actualDeparture ?? null,
      done: stop.status === "Completed",
    }));

    const nextIndex = stops.findIndex((stop) => !stop.done);
    const projections = shipment.eta?.stops ?? [];
    let eta: TimelineEta | null = null;
    if (nextIndex !== -1) {
      const next = stops[nextIndex];
      const own = projections.find((p) => p.stopId === next.id && !p.arrived);
      const governing = shipment.eta?.stopId === next.id ? shipment.eta : null;
      const projected = own?.estimatedArrival ?? governing?.estimatedArrival ?? null;
      if (projected) {
        // The server's slack decides lateness, as the table's ETA column reads it.
        const slack = own?.slackMinutes ?? governing?.slackMinutes;
        eta = {
          stopIndex: nextIndex,
          at: projected,
          lateMinutes:
            slack != null
              ? Math.max(0, -slack)
              : Math.max(0, Math.round((projected - next.windowEnd) / 60)),
        };
      }
    }

    const first = stops[0];
    const last = stops[stops.length - 1];
    const start = Math.min(first.windowStart, first.arrivedAt ?? Number.POSITIVE_INFINITY);
    const end = Math.max(last.windowEnd, last.departedAt ?? 0, eta?.at ?? 0);
    const driveMinutes = projections.length
      ? Math.max(...projections.map((p) => p.driveMinutes ?? 0))
      : null;
    const firstLiveMove = (shipment.moves ?? []).find((move) => move.status !== "Canceled");

    loads.push({
      id: shipment.id,
      shipment,
      stage,
      stops,
      start,
      end,
      eta,
      driven: stage === "Delivered" ? "all" : first.done ? "underway" : "none",
      coverage: resolveCoverage(shipment),
      moveId: firstLiveMove?.id ?? null,
      driveMinutes: driveMinutes && driveMinutes > 0 ? driveMinutes : null,
    });
  }
  return loads;
}

export function overlapsRange(load: TimelineLoad, from: number, to: number): boolean {
  return load.start < to && load.end > from;
}

export type TimelineStageGroup = { stage: ActiveStage; loads: TimelineLoad[] };

/** Loads grouped by stage in the board's order: the latest load first, then by start. */
export function groupLoadsByStage(loads: readonly TimelineLoad[]): TimelineStageGroup[] {
  return ACTIVE_STAGES.map((stage) => ({
    stage,
    loads: loads
      .filter((load) => load.stage === stage)
      .sort((a, b) =>
        stage === "Late"
          ? (b.eta?.lateMinutes ?? 0) - (a.eta?.lateMinutes ?? 0) || a.start - b.start
          : a.start - b.start,
      ),
  })).filter((group) => group.loads.length > 0);
}

/* ------------------------------------------------------------------ lanes */

export type DriverLane = {
  id: string;
  name: string;
  initials: string;
  tractorId: string | null;
  tractorCode: string | null;
  /** From the capacity strip when it knows the driver; otherwise read per lane. */
  driveRemainingMs: number | null;
  loads: TimelineLoad[];
  packed: PackedLane<TimelineLoad>;
};

export type CarrierLane = {
  id: string;
  name: string;
  initials: string;
  detail: string | null;
  loads: TimelineLoad[];
  packed: PackedLane<TimelineLoad>;
};

export type CoverageLanes = {
  /** Uncovered and tendered loads: work still to place. */
  pool: TimelineLoad[];
  poolPacked: PackedLane<TimelineLoad>;
  drivers: DriverLane[];
  carriers: CarrierLane[];
};

function tractorIdOf(load: TimelineLoad): string | null {
  for (const move of load.shipment.moves ?? []) {
    if (move.status !== "Canceled" && move.assignment?.tractorId) return move.assignment.tractorId;
  }
  return null;
}

/**
 * One lane per driver and carrier hauling a load in the range, plus a lane for
 * every driver the capacity strip lists who has nothing here, so there is
 * somewhere to put work. Double-booked drivers lead, then the earliest busy,
 * then the free ones by name.
 */
export function buildCoverageLanes(
  loads: readonly TimelineLoad[],
  capacityDrivers: readonly CapacityUnit[],
): CoverageLanes {
  const pool: TimelineLoad[] = [];
  const drivers = new Map<string, Omit<DriverLane, "packed">>();
  const carriers = new Map<string, Omit<CarrierLane, "packed">>();

  for (const unit of capacityDrivers) {
    drivers.set(unit.id, {
      id: unit.id,
      name: unit.name,
      initials: unit.initials,
      tractorId: unit.tractorId ?? null,
      tractorCode: unit.unitLabel ?? null,
      driveRemainingMs: unit.driveRemainingMs ?? null,
      loads: [],
    });
  }

  for (const load of loads) {
    const coverage = load.coverage;
    if (coverage.kind === "uncovered" || coverage.kind === "tendered") {
      pool.push(load);
    } else if (coverage.kind === "driver") {
      let lane = drivers.get(coverage.id);
      if (!lane) {
        lane = {
          id: coverage.id,
          name: coverage.name,
          initials: coverage.initials,
          tractorId: null,
          tractorCode: coverage.detail,
          driveRemainingMs: null,
          loads: [],
        };
        drivers.set(coverage.id, lane);
      }
      lane.tractorId ??= tractorIdOf(load);
      lane.tractorCode ??= coverage.detail;
      lane.loads.push(load);
    } else {
      let lane = carriers.get(coverage.id);
      if (!lane) {
        lane = {
          id: coverage.id,
          name: coverage.name,
          initials: coverage.initials,
          detail: coverage.detail,
          loads: [],
        };
        carriers.set(coverage.id, lane);
      }
      lane.loads.push(load);
    }
  }

  // Each lane is packed once; sorting reads the result instead of packing per comparison.
  const firstStart = (lane: { loads: TimelineLoad[] }) =>
    lane.loads.reduce((min, l) => Math.min(min, l.start), Number.POSITIVE_INFINITY);
  const driverLanes = [...drivers.values()].map((lane) => ({
    lane: { ...lane, packed: packLane(lane.loads) },
    first: firstStart(lane),
  }));
  const carrierLanes = [...carriers.values()].map((lane) => ({
    lane: { ...lane, packed: packLane(lane.loads, { flagOverlaps: false }) },
    first: firstStart(lane),
  }));
  const doubleBooked = (lane: DriverLane) => (lane.packed.conflicts.size > 0 ? 0 : 1);

  return {
    pool: pool.sort((a, b) => a.start - b.start),
    poolPacked: packLane(pool, { flagOverlaps: false }),
    drivers: driverLanes
      .sort(
        (a, b) =>
          doubleBooked(a.lane) - doubleBooked(b.lane) ||
          a.first - b.first ||
          a.lane.name.localeCompare(b.lane.name),
      )
      .map(({ lane }) => lane),
    carriers: carrierLanes
      .sort((a, b) => a.first - b.first || a.lane.name.localeCompare(b.lane.name))
      .map(({ lane }) => lane),
  };
}

/* ------------------------------------------------------------------ assigning */

export type AssignCheck = {
  /** A driver without a tractor cannot be dispatched; the assignment is refused. */
  blocked: boolean;
  /** The driver's other loads whose time overlaps this one. */
  overlaps: TimelineLoad[];
  /** Set when the load needs more driving than the driver has left today. */
  driveShort: { neededMinutes: number; remainingMs: number } | null;
};

/**
 * What a dispatcher should know before putting `load` on `lane`'s driver, at
 * `now` (Unix seconds). Drive time is checked only for a load that picks up
 * inside the driver's current duty window.
 */
export function checkAssignment(
  load: TimelineLoad,
  lane: DriverLane,
  driveRemainingMs: number | null,
  now: number,
): AssignCheck {
  const overlaps = lane.loads.filter(
    (other) => other.id !== load.id && other.start < load.end && load.start < other.end,
  );
  const neededMinutes = load.driveMinutes;
  const inDutyWindow = load.start - now < DUTY_WINDOW_SECONDS;
  const driveShort =
    inDutyWindow &&
    neededMinutes != null &&
    driveRemainingMs != null &&
    neededMinutes * 60_000 > driveRemainingMs
      ? { neededMinutes, remainingMs: driveRemainingMs }
      : null;
  return { blocked: !lane.tractorId, overlaps, driveShort };
}

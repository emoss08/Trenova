import { listShipmentsGraphQL } from "@/lib/graphql/shipment";
import { getShipmentEtaTone, type ShipmentEtaTone } from "@/lib/shipment-utils";
import type { FieldFilter, FilterGroup } from "@/types/data-table";
import type { Assignment, Shipment, ShipmentMove, Stop } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import type { TimelineSort } from "../url-state";
import { packLanes, type TimeRange } from "./time-scale";

/**
 * The timeline renders every shipment with stop activity inside the visible
 * window in a single pass (no pagination), so the fetch is capped. When the
 * server reports more matches than the cap the view surfaces a truncation
 * notice instead of silently dropping loads.
 */
const TIMELINE_FETCH_LIMIT = 250;

export const UNASSIGNED_ROW_KEY = "__unassigned__";

/**
 * Dwell thresholds mirror the industry's typical two-hour free time at a
 * facility: flag as "watch" when a driver has been sitting long enough that
 * detention is approaching, and "critical" once the free-time window is blown.
 */
export const DWELL_WATCH_SECONDS = 90 * 60;
export const DWELL_CRITICAL_SECONDS = 120 * 60;

export type TimelineStopMarker = {
  id: string;
  time: number;
  type: Stop["type"];
  status: Stop["status"];
  locationCode: string;
  locationName: string;
  scheduledStart: number | null;
  scheduledEnd: number | null;
  actualArrival: number | null;
  actualDeparture: number | null;
};

export type TimelineDwell = {
  stopId: string;
  locationCode: string;
  locationName: string;
  seconds: number;
  severity: "watch" | "critical";
};

export type TimelineBar = {
  moveId: string;
  shipment: Shipment;
  move: ShipmentMove;
  assignment: Assignment | null;
  start: number;
  end: number;
  tone: ShipmentEtaTone;
  isCanceled: boolean;
  stops: TimelineStopMarker[];
  laneIndex: number;
  dwell: TimelineDwell | null;
  hasOverlap: boolean;
};

export type TimelineRowStats = {
  late: number;
  watch: number;
  dwelling: number;
  overlaps: number;
};

export type TimelineRowAlert = "late" | "watch" | null;

export type TimelineRow = {
  key: string;
  workerName: string;
  workerProfilePicUrl: string | null;
  equipmentCodes: string[];
  bars: TimelineBar[];
  laneCount: number;
  stats: TimelineRowStats;
  alert: TimelineRowAlert;
};

export type TimelineExceptionSummary = {
  late: number;
  watch: number;
  dwelling: number;
  overlaps: number;
  unassigned: number;
};

export type TimelineFocus = keyof TimelineExceptionSummary;

export type TimelineData = {
  rows: TimelineRow[];
  unassignedRow: TimelineRow | null;
  barCount: number;
  shipmentCount: number;
  totalCount: number;
  truncated: boolean;
  exceptions: TimelineExceptionSummary;
};

/**
 * Rows come out of buildTimelineData sorted by worker name; the other modes
 * re-rank without mutating the source array. "Exceptions first" weighs a late
 * load above a dwelling one above a plain watch so the hottest drivers surface
 * at the top of the board.
 */
export function sortTimelineRows(rows: TimelineRow[], sort: TimelineSort): TimelineRow[] {
  if (sort === "name") return rows;
  const sorted = [...rows];
  if (sort === "loads") {
    sorted.sort(
      (a, b) => b.bars.length - a.bars.length || a.workerName.localeCompare(b.workerName),
    );
    return sorted;
  }
  const score = (row: TimelineRow) =>
    row.stats.late * 100 + row.stats.dwelling * 40 + row.stats.overlaps * 20 + row.stats.watch * 10;
  sorted.sort((a, b) => score(b) - score(a) || a.workerName.localeCompare(b.workerName));
  return sorted;
}

export function barMatchesFocus(bar: TimelineBar, focus: TimelineFocus): boolean {
  switch (focus) {
    case "late":
      return bar.tone === "late" && !bar.isCanceled;
    case "watch":
      return bar.tone === "watch" && !bar.isCanceled;
    case "dwelling":
      return bar.dwell !== null;
    case "overlaps":
      return bar.hasOverlap;
    case "unassigned":
      return !bar.assignment && !bar.isCanceled;
  }
}

type UseTimelineDataParams = {
  range: TimeRange;
  fieldFilters: FieldFilter[];
  filterGroups: FilterGroup[] | undefined;
  query: string;
  now: number;
  enabled: boolean;
};

export function useTimelineData({
  range,
  fieldFilters,
  filterGroups,
  query,
  now,
  enabled,
}: UseTimelineDataParams) {
  const dataQuery = useQuery({
    queryKey: [
      "shipment-list",
      "timeline",
      { start: range.start, end: range.end },
      fieldFilters,
      filterGroups,
      query,
    ],
    queryFn: () =>
      listShipmentsGraphQL({
        limit: TIMELINE_FETCH_LIMIT,
        query,
        fieldFilters,
        filterGroups,
        activityWindowStart: range.start,
        activityWindowEnd: range.end,
      }),
    placeholderData: (prev) => prev,
    enabled,
  });

  const queryData = dataQuery.data;
  const data = useMemo<TimelineData>(
    () =>
      buildTimelineData(
        (queryData?.results ?? []) as Shipment[],
        queryData?.count ?? 0,
        range,
        now,
      ),
    [queryData, range, now],
  );

  return { dataQuery, data };
}

export function buildTimelineData(
  shipments: Shipment[],
  totalCount: number,
  range: TimeRange,
  nowSeconds: number,
): TimelineData {
  const rowsByKey = new Map<string, TimelineRow>();
  let barCount = 0;

  for (const shipment of shipments) {
    if (!shipment.moves?.length) continue;
    const tone = getShipmentEtaTone(shipment);
    const shipmentCanceled = shipment.status === "Canceled";

    for (const move of shipment.moves) {
      const moveCanceled = move.status === "Canceled";
      // A canceled move on a live shipment is a re-planned leg — hide it. When
      // the whole shipment is canceled (and filters chose to show it) keep the
      // bars visible in a muted state so the row doesn't silently vanish.
      if (moveCanceled && !shipmentCanceled) continue;

      const span = getMoveSpan(move);
      if (!span || span.start >= range.end || span.end <= range.start) continue;

      const assignment = move.assignment ?? null;
      const worker = assignment?.primaryWorker;
      const key = worker?.id ?? UNASSIGNED_ROW_KEY;
      let row = rowsByKey.get(key);
      if (!row) {
        row = {
          key,
          workerName: getWorkerDisplayName(worker),
          workerProfilePicUrl: worker?.profilePicUrl ?? null,
          equipmentCodes: [],
          bars: [],
          laneCount: 1,
          stats: { late: 0, watch: 0, dwelling: 0, overlaps: 0 },
          alert: null,
        };
        rowsByKey.set(key, row);
      }

      const tractorCode = assignment?.tractor?.code;
      if (tractorCode && !row.equipmentCodes.includes(tractorCode)) {
        row.equipmentCodes.push(tractorCode);
      }

      const isCanceled = shipmentCanceled || moveCanceled;
      row.bars.push({
        moveId: move.id ?? `${shipment.id}-${move.sequence}`,
        shipment,
        move,
        assignment,
        start: span.start,
        end: span.end,
        tone: isCanceled ? "pending" : tone,
        isCanceled,
        stops: span.stops,
        laneIndex: 0,
        dwell: isCanceled ? null : getBarDwell(span.stops, nowSeconds),
        hasOverlap: false,
      });
      barCount++;
    }
  }

  const exceptions: TimelineExceptionSummary = {
    late: 0,
    watch: 0,
    dwelling: 0,
    overlaps: 0,
    unassigned: 0,
  };

  for (const row of rowsByKey.values()) {
    row.bars.sort((a, b) => a.start - b.start || a.end - b.end);
    const packing = packLanes(row.bars);
    for (let i = 0; i < row.bars.length; i++) {
      row.bars[i].laneIndex = packing.laneByIndex[i];
    }
    row.laneCount = packing.laneCount;

    if (row.key !== UNASSIGNED_ROW_KEY) markOverlaps(row.bars);

    let hasCriticalDwell = false;
    for (const bar of row.bars) {
      if (bar.isCanceled) continue;
      if (bar.tone === "late") row.stats.late++;
      if (bar.tone === "watch") row.stats.watch++;
      if (bar.dwell) {
        row.stats.dwelling++;
        if (bar.dwell.severity === "critical") hasCriticalDwell = true;
      }
      if (bar.hasOverlap) row.stats.overlaps++;
    }

    if (row.stats.late > 0 || hasCriticalDwell) {
      row.alert = "late";
    } else if (row.stats.watch > 0 || row.stats.dwelling > 0 || row.stats.overlaps > 0) {
      row.alert = "watch";
    }

    exceptions.late += row.stats.late;
    exceptions.watch += row.stats.watch;
    exceptions.dwelling += row.stats.dwelling;
    exceptions.overlaps += row.stats.overlaps;
  }

  const unassignedRow = rowsByKey.get(UNASSIGNED_ROW_KEY) ?? null;
  rowsByKey.delete(UNASSIGNED_ROW_KEY);

  if (unassignedRow) {
    exceptions.unassigned = unassignedRow.bars.filter((bar) => !bar.isCanceled).length;
    if (exceptions.unassigned > 0 && !unassignedRow.alert) unassignedRow.alert = "watch";
  }

  const rows = [...rowsByKey.values()].sort((a, b) => a.workerName.localeCompare(b.workerName));

  return {
    rows,
    unassignedRow,
    barCount,
    shipmentCount: shipments.length,
    totalCount,
    truncated: totalCount > shipments.length,
    exceptions,
  };
}

/**
 * Two live moves occupying the same clock time on one driver is a probable
 * double-booking. Bars arrive sorted by start, so a linear forward scan is
 * enough to flag every overlapping pair.
 */
function markOverlaps(bars: TimelineBar[]): void {
  for (let i = 0; i < bars.length; i++) {
    if (bars[i].isCanceled) continue;
    for (let j = i + 1; j < bars.length && bars[j].start < bars[i].end; j++) {
      if (bars[j].isCanceled) continue;
      bars[i].hasOverlap = true;
      bars[j].hasOverlap = true;
    }
  }
}

function getBarDwell(stops: TimelineStopMarker[], nowSeconds: number): TimelineDwell | null {
  let dwell: TimelineDwell | null = null;

  for (const stop of stops) {
    if (!stop.actualArrival || stop.actualDeparture) continue;
    if (stop.status === "Completed" || stop.status === "Canceled") continue;
    const seconds = nowSeconds - stop.actualArrival;
    if (seconds < DWELL_WATCH_SECONDS) continue;
    if (dwell && dwell.seconds >= seconds) continue;
    dwell = {
      stopId: stop.id,
      locationCode: stop.locationCode,
      locationName: stop.locationName,
      seconds,
      severity: seconds >= DWELL_CRITICAL_SECONDS ? "critical" : "watch",
    };
  }

  return dwell;
}

function getWorkerDisplayName(
  worker: Assignment["primaryWorker"] | undefined,
): string {
  // wholeName is a scanonly generated column the server doesn't always select
  // on nested assignment loads — fall back to composing it, like DriverCell.
  const wholeName = worker?.wholeName?.trim();
  if (wholeName) return wholeName;
  const composed = [worker?.firstName, worker?.lastName].filter(Boolean).join(" ").trim();
  return composed || "Unassigned";
}

type MoveSpan = {
  start: number;
  end: number;
  stops: TimelineStopMarker[];
};

function getMoveSpan(move: ShipmentMove): MoveSpan | null {
  if (!move.stops?.length) return null;

  let start = Number.POSITIVE_INFINITY;
  let end = Number.NEGATIVE_INFINITY;
  const markers: TimelineStopMarker[] = [];

  for (const stop of move.stops) {
    const scheduledStart = stop.scheduledWindowStart > 0 ? stop.scheduledWindowStart : null;
    const stopStart = stop.actualArrival ?? scheduledStart;
    if (!stopStart) continue;
    const stopEnd =
      stop.actualDeparture ?? stop.scheduledWindowEnd ?? scheduledStart ?? stop.actualArrival;

    start = Math.min(start, stopStart);
    end = Math.max(end, stopEnd ?? stopStart);
    markers.push({
      id: stop.id ?? `${move.id}-${stop.sequence}`,
      time: stopStart,
      type: stop.type,
      status: stop.status,
      locationCode: stop.location?.code ?? "—",
      locationName: stop.location?.name ?? stop.addressLine ?? "Unknown location",
      scheduledStart,
      scheduledEnd: stop.scheduledWindowEnd ?? null,
      actualArrival: stop.actualArrival ?? null,
      actualDeparture: stop.actualDeparture ?? null,
    });
  }

  if (!Number.isFinite(start) || !Number.isFinite(end)) return null;

  return { start, end: Math.max(end, start), stops: markers };
}

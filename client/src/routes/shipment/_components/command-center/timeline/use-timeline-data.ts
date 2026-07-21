import { listShipmentsGraphQL } from "@/lib/graphql/shipment";
import { getShipmentEtaTone, type ShipmentEtaTone } from "@/lib/shipment-utils";
import type { FieldFilter, FilterGroup } from "@/types/data-table";
import type { Assignment, Shipment, ShipmentMove, Stop } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { packLanes, type TimeRange } from "./time-scale";

/**
 * The timeline renders every shipment with stop activity inside the visible
 * window in a single pass (no pagination), so the fetch is capped. When the
 * server reports more matches than the cap the view surfaces a truncation
 * notice instead of silently dropping loads.
 */
const TIMELINE_FETCH_LIMIT = 250;

export const UNASSIGNED_ROW_KEY = "__unassigned__";

export type TimelineStopMarker = {
  id: string;
  time: number;
  type: Stop["type"];
  status: Stop["status"];
  locationCode: string;
  locationName: string;
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
};

export type TimelineRow = {
  key: string;
  workerName: string;
  workerProfilePicUrl: string | null;
  equipmentCodes: string[];
  bars: TimelineBar[];
  laneCount: number;
};

export type TimelineData = {
  rows: TimelineRow[];
  unassignedRow: TimelineRow | null;
  barCount: number;
  shipmentCount: number;
  totalCount: number;
  truncated: boolean;
};

type UseTimelineDataParams = {
  range: TimeRange;
  fieldFilters: FieldFilter[];
  filterGroups: FilterGroup[] | undefined;
  query: string;
  enabled: boolean;
};

export function useTimelineData({
  range,
  fieldFilters,
  filterGroups,
  query,
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
      ),
    [queryData, range],
  );

  return { dataQuery, data };
}

export function buildTimelineData(
  shipments: Shipment[],
  totalCount: number,
  range: TimeRange,
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
        };
        rowsByKey.set(key, row);
      }

      const tractorCode = assignment?.tractor?.code;
      if (tractorCode && !row.equipmentCodes.includes(tractorCode)) {
        row.equipmentCodes.push(tractorCode);
      }

      row.bars.push({
        moveId: move.id ?? `${shipment.id}-${move.sequence}`,
        shipment,
        move,
        assignment,
        start: span.start,
        end: span.end,
        tone: shipmentCanceled || moveCanceled ? "pending" : tone,
        isCanceled: shipmentCanceled || moveCanceled,
        stops: span.stops,
        laneIndex: 0,
      });
      barCount++;
    }
  }

  for (const row of rowsByKey.values()) {
    row.bars.sort((a, b) => a.start - b.start || a.end - b.end);
    const packing = packLanes(row.bars);
    for (let i = 0; i < row.bars.length; i++) {
      row.bars[i].laneIndex = packing.laneByIndex[i];
    }
    row.laneCount = packing.laneCount;
  }

  const unassignedRow = rowsByKey.get(UNASSIGNED_ROW_KEY) ?? null;
  rowsByKey.delete(UNASSIGNED_ROW_KEY);

  const rows = [...rowsByKey.values()].sort((a, b) => a.workerName.localeCompare(b.workerName));

  return {
    rows,
    unassignedRow,
    barCount,
    shipmentCount: shipments.length,
    totalCount,
    truncated: totalCount > shipments.length,
  };
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
    });
  }

  if (!Number.isFinite(start) || !Number.isFinite(end)) return null;

  return { start, end: Math.max(end, start), stops: markers };
}

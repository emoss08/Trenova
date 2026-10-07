import type { ShipmentBoardGrouping, ShipmentStage } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  formatISODateMedium,
  fromISODateString,
  fromUserWallClock,
  toISODateString,
  toUserWallClock,
} from "@trenova/shared/lib/date";
import type {
  DataTableGroup,
  DataTableGroupKey,
  DataTableGroupScope,
  FieldFilter,
  FilterGroup,
  SortField,
} from "@trenova/shared/types/data-table";
import type { Shipment, ShipmentMove, Stop } from "@trenova/shared/types/shipment";
import { addDays } from "date-fns";
import type { ShipmentBoardGroup } from "@/lib/graphql/shipment-board";
import { STAGE_GROUP_FIELD, wholeDollars } from "./stage";

export const BOARD_GROUPINGS = [
  "stage",
  "shipDate",
  "deliveryDate",
  "customer",
  "owner",
  "none",
] as const;
export type BoardGrouping = (typeof BOARD_GROUPINGS)[number];
export type ServerBoardGrouping = Exclude<BoardGrouping, "stage" | "none">;

/** The value a group has no date or no owner under, on both client and server. */
export const NO_GROUP_VALUE = "";

export const SERVER_GROUPING: Record<ServerBoardGrouping, ShipmentBoardGrouping> = {
  shipDate: "ShipDate",
  deliveryDate: "DeliveryDate",
  customer: "Customer",
  owner: "Owner",
};

export function isServerGrouping(grouping: BoardGrouping): grouping is ServerBoardGrouping {
  return grouping !== "stage" && grouping !== "none";
}

const GROUPING_LABEL_ENTRIES: ReadonlyArray<{ value: BoardGrouping; label: string }> = [
  { value: "stage", label: "Status" },
  { value: "shipDate", label: "Ship date" },
  { value: "deliveryDate", label: "Delivery date" },
  { value: "customer", label: "Customer" },
  { value: "owner", label: "Owner" },
  { value: "none", label: "No grouping" },
];

/** English labels, translated where they are rendered. */
export const GROUPING_LABELS = Object.fromEntries(
  GROUPING_LABEL_ENTRIES.map((entry) => [entry.value, entry.label]),
) as Record<BoardGrouping, string>;

const SHIP_DATE_FIELD = "shipperStop.scheduledWindowStart";
const DELIVERY_DATE_FIELD = "consigneeStop.scheduledWindowStart";

type GroupSort = { field: string; tieBreakers?: SortField[] };

const GROUP_SORT: Record<Exclude<BoardGrouping, "none">, GroupSort> = {
  stage: { field: STAGE_GROUP_FIELD },
  shipDate: { field: SHIP_DATE_FIELD },
  deliveryDate: { field: DELIVERY_DATE_FIELD },
  customer: {
    field: "customer.name",
    tieBreakers: [{ field: "customerId", direction: "asc" }],
  },
  owner: {
    field: "owner.name",
    tieBreakers: [{ field: "ownerId", direction: "asc" }],
  },
};

export function groupSort(grouping: Exclude<BoardGrouping, "none">): GroupSort {
  return GROUP_SORT[grouping];
}

const ORIGIN_TYPES = new Set(["Pickup", "SplitPickup"]);
const DESTINATION_TYPES = new Set(["Delivery", "SplitDelivery"]);

type EndStopOrder = 1 | -1;

/*
 * Mirrors the server's FirstShipperStop and LastConsigneeStop: earliest (or
 * latest) move, then stop within it, the stop id settling a tie by code
 * point. A row's group must be the one the server sorted and counted it in.
 */
function endStop(
  moves: readonly ShipmentMove[] | null | undefined,
  types: ReadonlySet<string>,
  order: EndStopOrder,
): Stop | null {
  let best: Stop | null = null;
  let bestMove = 0;
  for (const move of moves ?? []) {
    for (const stop of move.stops ?? []) {
      if (!stop || !types.has(stop.type ?? "")) continue;
      if (!best || compareEndStops(move, stop, bestMove, best) * order < 0) {
        best = stop;
        bestMove = move.sequence ?? 0;
      }
    }
  }
  return best;
}

function compareEndStops(move: ShipmentMove, stop: Stop, bestMove: number, best: Stop): number {
  const moveSeq = move.sequence ?? 0;
  if (moveSeq !== bestMove) return moveSeq < bestMove ? -1 : 1;
  const stopSeq = stop.sequence ?? 0;
  const bestSeq = best.sequence ?? 0;
  if (stopSeq !== bestSeq) return stopSeq < bestSeq ? -1 : 1;
  const id = stop.id ?? "";
  const bestId = best.id ?? "";
  if (id === bestId) return 0;
  return id < bestId ? -1 : 1;
}

export function shipperStop(row: Pick<Shipment, "moves">): Stop | null {
  return endStop(row.moves, ORIGIN_TYPES, 1);
}

export function consigneeStop(row: Pick<Shipment, "moves">): Stop | null {
  return endStop(row.moves, DESTINATION_TYPES, -1);
}

/** The `YYYY-MM-DD` day an instant falls on in the board's time zone. */
export function dayKey(unixSeconds: number | null | undefined, timezone: string): string {
  if (unixSeconds === null || unixSeconds === undefined) return NO_GROUP_VALUE;
  const wallClock = toUserWallClock(unixSeconds, timezone);
  return wallClock ? toISODateString(wallClock) : NO_GROUP_VALUE;
}

/** The instants a local day covers, `[start, end)`, so a DST day is 23 or 25 hours. */
export function dayBounds(key: string, timezone: string): { start: number; end: number } | null {
  const day = fromISODateString(key);
  if (!day) return null;
  const start = fromUserWallClock(day, timezone);
  const end = fromUserWallClock(addDays(day, 1), timezone);
  return start === undefined || end === undefined ? null : { start, end };
}

export function boardGroupKey(
  grouping: Exclude<BoardGrouping, "none">,
  row: Shipment,
  timezone: string,
  stageRankOf: (stage: ShipmentStage | null | undefined) => DataTableGroupKey,
): DataTableGroupKey {
  switch (grouping) {
    case "stage":
      return stageRankOf(row.stage as ShipmentStage | undefined);
    case "shipDate":
      return dayKey(shipperStop(row)?.scheduledWindowStart, timezone);
    case "deliveryDate":
      return dayKey(consigneeStop(row)?.scheduledWindowStart, timezone);
    case "customer":
      return row.customerId ?? NO_GROUP_VALUE;
    case "owner":
      return row.ownerId ?? NO_GROUP_VALUE;
  }
}

/** URL keys are strings; the stage field is a rank, so stage keys go back to numbers. */
export function parseCollapsedKeys(
  grouping: BoardGrouping,
  keys: readonly string[],
): DataTableGroupKey[] {
  if (grouping !== "stage") return [...keys];
  return keys.map(Number).filter((rank) => Number.isInteger(rank));
}

function dayScope(field: string, keys: readonly DataTableGroupKey[], timezone: string) {
  const fieldFilters: FieldFilter[] = [];
  const filterGroups: FilterGroup[] = [];
  for (const key of keys) {
    if (key === NO_GROUP_VALUE) {
      fieldFilters.push({ field, operator: "isnotnull", value: null });
      continue;
    }
    const bounds = dayBounds(String(key), timezone);
    if (!bounds) continue;
    filterGroups.push({
      filters: [
        { field, operator: "lt", value: bounds.start },
        { field, operator: "gte", value: bounds.end },
        { field, operator: "isnull", value: null },
      ],
    });
  }
  return { fieldFilters, filterGroups };
}

function ownerScope(keys: readonly DataTableGroupKey[]): DataTableGroupScope {
  const ids = keys.filter((key) => key !== NO_GROUP_VALUE).map(String);
  const withoutOwner = keys.includes(NO_GROUP_VALUE);
  const fieldFilters: FieldFilter[] = withoutOwner
    ? [{ field: "ownerId", operator: "isnotnull", value: null }]
    : [];
  const filterGroups: FilterGroup[] = [];
  if (ids.length > 0) {
    const notIn: FieldFilter = { field: "ownerId", operator: "notin", value: ids };
    if (withoutOwner) {
      fieldFilters.push(notIn);
    } else {
      filterGroups.push({
        filters: [notIn, { field: "ownerId", operator: "isnull", value: null }],
      });
    }
  }
  return { fieldFilters, filterGroups };
}

/**
 * How the server drops collapsed groups. Stage and customer keys are the
 * field's own values, so the table's default `notin` serves them; a day is a
 * range of instants and "no owner" is a null, so those say it themselves.
 */
export function collapsedScopeFor(
  grouping: Exclude<BoardGrouping, "none">,
  timezone: string,
): ((keys: readonly DataTableGroupKey[]) => DataTableGroupScope) | undefined {
  switch (grouping) {
    case "shipDate":
      return (keys) => dayScope(SHIP_DATE_FIELD, keys, timezone);
    case "deliveryDate":
      return (keys) => dayScope(DELIVERY_DATE_FIELD, keys, timezone);
    case "customer":
      return (keys) => ({
        fieldFilters: [{ field: "customerId", operator: "notin", value: keys.map(String) }],
        filterGroups: [],
      });
    case "owner":
      return ownerScope;
    case "stage":
      return undefined;
  }
}

function dayLabel(key: string, t: TranslateFn, today: string): string {
  const date = fromISODateString(key);
  const todayDate = fromISODateString(today);
  if (date && todayDate) {
    const offset = Math.round((date.getTime() - todayDate.getTime()) / 86_400_000);
    if (offset === 0) return t("Today · {0}", formatISODateMedium(key));
    if (offset === 1) return t("Tomorrow · {0}", formatISODateMedium(key));
    if (offset === -1) return t("Yesterday · {0}", formatISODateMedium(key));
  }
  return formatISODateMedium(key, key);
}

type BoardGroupHeadersParams = {
  grouping: ServerBoardGrouping;
  groups: readonly ShipmentBoardGroup[];
  t: TranslateFn;
  timezone: string;
  now?: Date;
};

/** Headers for a server grouping, in the order the server sorted its rows. */
export function boardGroupHeaders({
  grouping,
  groups,
  t,
  timezone,
  now = new Date(),
}: BoardGroupHeadersParams): DataTableGroup[] {
  const today = dayKey(Math.floor(now.getTime() / 1000), timezone);
  return groups
    .filter((group) => group.count > 0)
    .map((group) => ({
      key: group.key,
      label: groupLabel(grouping, group, t, today),
      count: group.count,
      aggregate: wholeDollars(group.revenue),
    }));
}

function groupLabel(
  grouping: ServerBoardGrouping,
  group: ShipmentBoardGroup,
  t: TranslateFn,
  today: string,
): string {
  switch (grouping) {
    case "shipDate":
      return group.key === NO_GROUP_VALUE ? t("No ship date") : dayLabel(group.key, t, today);
    case "deliveryDate":
      return group.key === NO_GROUP_VALUE ? t("No delivery date") : dayLabel(group.key, t, today);
    case "customer":
      return group.label || t("Unnamed customer");
    case "owner":
      return group.key === NO_GROUP_VALUE ? t("No owner") : group.label || t("Former user");
  }
}

import {
  fromISODateString,
  fromUserWallClock,
  toISODateString,
  toUserWallClock,
} from "@trenova/shared/lib/date";
import type {
  DataTableGroupKey,
  DataTableGroupScope,
  FieldFilter,
  FilterGroup,
} from "@trenova/shared/types/data-table";
import { addDays } from "date-fns";

/** The key of rows with no instant: a group of their own. */
export const NO_DAY = "";

/** The local day an instant falls on, YYYY-MM-DD in the time zone, as a table group's key. */
export function dayKey(unixSeconds: number | null | undefined, timezone: string): string {
  if (unixSeconds === null || unixSeconds === undefined) return NO_DAY;
  const wallClock = toUserWallClock(unixSeconds, timezone);
  return wallClock ? toISODateString(wallClock) : NO_DAY;
}

/** The instants a local day covers, `[start, end)`, so a DST day is 23 or 25 hours. */
export function dayBounds(key: string, timezone: string): { start: number; end: number } | null {
  const day = fromISODateString(key);
  if (!day) return null;
  const start = fromUserWallClock(day, timezone);
  const end = fromUserWallClock(addDays(day, 1), timezone);
  return start === undefined || end === undefined ? null : { start, end };
}

/**
 * The filters that drop collapsed day groups from a table grouped by the day of `field`: each
 * collapsed day keeps only rows before or after it, and a collapsed "no day" group keeps only
 * rows that have one.
 */
export function dayGroupScope(
  field: string,
  keys: readonly DataTableGroupKey[],
  timezone: string,
): DataTableGroupScope {
  const fieldFilters: FieldFilter[] = [];
  const filterGroups: FilterGroup[] = [];
  for (const key of keys) {
    if (key === NO_DAY) {
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

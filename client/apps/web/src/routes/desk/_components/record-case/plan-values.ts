import { isOpaqueId } from "@/components/assistant/proposal-presenters";
import type { PreviewFieldChange } from "@/lib/graphql/agent-preview";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";

/** A date and time with no zone: the wall clock where the record is, as tools preview a stop's window. */
const WALL_CLOCK = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/;

/**
 * A wall-clock time as a person reads one ("Nov 10, 1:00 PM"), kept at the
 * hour it names: it is the stop's own clock, so it is never moved into the
 * reader's zone. Anything else is returned as it came.
 */
export function readableWallClock(value: unknown): unknown {
  if (typeof value !== "string") {
    return value;
  }
  const parts = WALL_CLOCK.exec(value.trim());
  if (!parts) {
    return value;
  }
  const [, year, month, day, hour, minute] = parts;
  const instant = Date.UTC(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
  );

  return formatUnixDateTimeShort(Math.floor(instant / 1000), { timezone: "UTC", fallback: value });
}

/** A step's value with its wall-clock times made readable, for the plan card's rows. */
export function readableField(field: PreviewFieldChange): PreviewFieldChange {
  return {
    ...field,
    before: readableWallClock(field.before),
    after: readableWallClock(field.after),
    proposedBefore: readableWallClock(field.proposedBefore),
  };
}

/**
 * Whether a previewed value is only an id nobody can read: a record's key
 * with no name beside it. The plan card leaves those out, since a person
 * approves what changes, never a key.
 */
export function isBareId(field: PreviewFieldChange): boolean {
  if (field.beforeRef || field.afterRef || field.withheld) {
    return false;
  }
  const values = [field.before, field.after].filter(
    (value) => value !== null && value !== undefined && value !== "",
  );

  return (
    values.length > 0 && values.every((value) => typeof value === "string" && isOpaqueId(value))
  );
}

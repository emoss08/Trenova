import type { TranslateFn } from "@trenova/shared/i18n/runtime";

/**
 * How a waiting message stands against its response target.
 *
 * The server sets `dueAt` from the organization's target for the message's
 * kind; the list only says how far off it is. Under half an hour left is a
 * warning, past it is over, and the list sorts and groups on the same lines.
 */
export type DueState = "over" | "soon" | "later";

export type DueSpan = {
  state: DueState;
  amount: number;
  unit: "m" | "h" | "d";
};

/** Inside this many minutes a message is about to be over target. */
export const SOON_MINUTES = 30;
/** The "due within the hour" group. */
export const HOUR_MINUTES = 60;

const MINUTES_PER_HOUR = 60;
const MINUTES_PER_DAY = 1440;

export function dueSpan(dueAt: number, now: number): DueSpan {
  const minutes = Math.floor((dueAt - now) / 60);
  const left = Math.abs(minutes);
  const state: DueState = minutes < 0 ? "over" : minutes < SOON_MINUTES ? "soon" : "later";

  if (left < MINUTES_PER_HOUR) {
    return { state, amount: left, unit: "m" };
  }
  if (left < MINUTES_PER_DAY) {
    return { state, amount: Math.floor(left / MINUTES_PER_HOUR), unit: "h" };
  }

  return { state, amount: Math.floor(left / MINUTES_PER_DAY), unit: "d" };
}

/** "38m left", "3h over", "1d over". */
export function dueText(t: TranslateFn, span: DueSpan): string {
  if (span.state === "over") {
    switch (span.unit) {
      case "m":
        return t("{0}m over", span.amount);
      case "h":
        return t("{0}h over", span.amount);
      case "d":
        return t("{0}d over", span.amount);
    }
  }
  switch (span.unit) {
    case "m":
      return t("{0}m left", span.amount);
    case "h":
      return t("{0}h left", span.amount);
    case "d":
      return t("{0}d left", span.amount);
  }
}

export type DueGroup = "over" | "hour" | "later";

/** Which group a message falls in when the folder is read due first. */
export function dueGroupOf(dueAt: number | null, now: number): DueGroup {
  if (dueAt === null) {
    return "later";
  }
  const minutes = (dueAt - now) / 60;
  if (minutes < 0) {
    return "over";
  }

  return minutes < HOUR_MINUTES ? "hour" : "later";
}

export type Grouped<G, T> = { group: G; items: T[] };

/**
 * A folder read due first, split into over target, due within the hour and
 * later. The server's order is kept inside each group; an empty group is left
 * out rather than drawn as a heading over nothing.
 */
export function groupByDue<T extends { dueAt: number | null; waiting: boolean }>(
  items: readonly T[],
  now: number,
): Grouped<DueGroup, T>[] {
  const buckets: Record<DueGroup, T[]> = { over: [], hour: [], later: [] };
  for (const item of items) {
    buckets[item.waiting ? dueGroupOf(item.dueAt, now) : "later"].push(item);
  }

  return (["over", "hour", "later"] as const)
    .filter((group) => buckets[group].length > 0)
    .map((group) => ({ group, items: buckets[group] }));
}

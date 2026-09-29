import type {
  CaptureBatchFilter,
  CaptureBatchSort,
  CaptureBatchStatus,
  CaptureSource,
} from "@/lib/graphql/capture";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { startOfDay, subDays } from "date-fns";

/**
 * The queue's views, by what a person can do with a stack. There is one queue;
 * a view is a set of statuses over it, never a second list.
 */
export const INTAKE_VIEWS = ["waiting", "working", "filed", "closed", "all"] as const;

export type IntakeView = (typeof INTAKE_VIEWS)[number];

export const INTAKE_SORTS = [
  "Newest",
  "Oldest",
  "ExpiringSoonest",
  "MostPages",
] as const satisfies readonly CaptureBatchSort[];

/** How far back a stack may have arrived, counted in whole days. */
export const INTAKE_RECEIVED = ["any", "today", "week", "month"] as const;

export type IntakeReceived = (typeof INTAKE_RECEIVED)[number];

export type IntakeFilter = {
  view: IntakeView;
  source: CaptureSource | null;
  mine: boolean;
  query: string;
  sort: CaptureBatchSort;
  received: IntakeReceived;
};

export const DEFAULT_INTAKE_FILTER: IntakeFilter = {
  view: "waiting",
  source: null,
  mine: false,
  query: "",
  sort: "Newest",
  received: "any",
};

export function viewStatuses(view: IntakeView): CaptureBatchStatus[] {
  switch (view) {
    case "waiting":
      return ["Ready", "PartiallyFiled"];
    case "working":
      return ["Receiving", "Sealed", "Processing"];
    case "filed":
      return ["Filed"];
    case "closed":
      return ["Discarded", "Expired", "Failed"];
    case "all":
      return [];
  }
}

export function viewLabel(t: TranslateFn, view: IntakeView): string {
  switch (view) {
    case "waiting":
      return t("To file");
    case "working":
      return t("Arriving");
    case "filed":
      return t("Filed");
    case "closed":
      return t("Discarded and expired");
    case "all":
      return t("Everything");
  }
}

export function sortLabel(t: TranslateFn, sort: CaptureBatchSort): string {
  switch (sort) {
    case "Newest":
      return t("Newest first");
    case "Oldest":
      return t("Oldest first");
    case "ExpiringSoonest":
      return t("Expiring soonest");
    case "MostPages":
      return t("Most pages");
  }
}

export function receivedLabel(t: TranslateFn, received: IntakeReceived): string {
  switch (received) {
    case "any":
      return t("Any time");
    case "today":
      return t("Today");
    case "week":
      return t("Last 7 days");
    case "month":
      return t("Last 30 days");
  }
}

/**
 * The start of the window, in Unix seconds, or null for any time. It starts
 * at midnight, so the request, and the cache entry it is kept under, stay the
 * same all day.
 */
export function receivedSince(received: IntakeReceived, now: number): number | null {
  const today = startOfDay(now);
  switch (received) {
    case "any":
      return null;
    case "today":
      return Math.floor(today.getTime() / 1000);
    case "week":
      return Math.floor(subDays(today, 6).getTime() / 1000);
    case "month":
      return Math.floor(subDays(today, 29).getTime() / 1000);
  }
}

function oneOf<T extends string>(value: string | null, options: readonly T[], fallback: T): T {
  return value !== null && (options as readonly string[]).includes(value) ? (value as T) : fallback;
}

/**
 * The queue's state as the address holds it, so a link to "my stacks that
 * are about to expire" opens exactly that. Anything the address does not say
 * or says wrongly falls back to the default rather than failing.
 */
export function parseIntakeFilter(params: URLSearchParams): IntakeFilter {
  return {
    view: oneOf(params.get("view"), INTAKE_VIEWS, DEFAULT_INTAKE_FILTER.view),
    source: oneOf<CaptureSource | "">(params.get("source"), ["Scan", "Print", ""], "") || null,
    mine: params.get("mine") === "1",
    query: params.get("q") ?? "",
    sort: oneOf(params.get("sort"), INTAKE_SORTS, DEFAULT_INTAKE_FILTER.sort),
    received: oneOf(params.get("received"), INTAKE_RECEIVED, DEFAULT_INTAKE_FILTER.received),
  };
}

/** Writes a filter into the address, leaving out what is the default. */
export function writeIntakeFilter(current: URLSearchParams, filter: IntakeFilter): URLSearchParams {
  const next = new URLSearchParams(current);
  const set = (key: string, value: string, fallback: string) => {
    if (value === fallback) {
      next.delete(key);
    } else {
      next.set(key, value);
    }
  };

  set("view", filter.view, DEFAULT_INTAKE_FILTER.view);
  set("source", filter.source ?? "", "");
  set("mine", filter.mine ? "1" : "", "");
  set("q", filter.query.trim(), "");
  set("sort", filter.sort, DEFAULT_INTAKE_FILTER.sort);
  set("received", filter.received, DEFAULT_INTAKE_FILTER.received);

  return next;
}

/** The request the queue sends for a filter. */
export function batchFilter(
  filter: IntakeFilter,
  now: number = Date.now(),
): Omit<CaptureBatchFilter, "after"> {
  return {
    statuses: viewStatuses(filter.view),
    source: filter.source,
    mine: filter.mine,
    query: filter.query.trim() === "" ? null : filter.query.trim(),
    sort: filter.sort,
    createdFrom: receivedSince(filter.received, now),
  };
}

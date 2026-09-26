import type {
  CaptureBatchFilter,
  CaptureBatchSort,
  CaptureBatchStatus,
  CaptureSource,
} from "@/lib/graphql/capture";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

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

export type IntakeFilter = {
  view: IntakeView;
  source: CaptureSource | null;
  mine: boolean;
  query: string;
  sort: CaptureBatchSort;
};

export const DEFAULT_INTAKE_FILTER: IntakeFilter = {
  view: "waiting",
  source: null,
  mine: false,
  query: "",
  sort: "Newest",
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

  return next;
}

/** The request the queue sends for a filter. */
export function batchFilter(filter: IntakeFilter): Omit<CaptureBatchFilter, "after"> {
  return {
    statuses: viewStatuses(filter.view),
    source: filter.source,
    mine: filter.mine,
    query: filter.query.trim() === "" ? null : filter.query.trim(),
    sort: filter.sort,
  };
}

import type { InsightCategory, InsightSeverity } from "@/types/insight";

/**
 * Which lifecycle a reader is asking about.
 *
 * This is deliberately a small closed set rather than the raw status enum. The
 * server distinguishes Resolved from Superseded — one condition went away, the
 * other was replaced by fresher numbers — and that difference is machinery, not
 * something a person browsing their findings needs to hold in their head.
 */
export type InsightStatusFilter = "active" | "dismissed" | "closed";

export const STATUS_FILTERS: InsightStatusFilter[] = ["active", "dismissed", "closed"];

export const CATEGORY_FILTERS: InsightCategory[] = [
  "ServiceQuality",
  "CashFlow",
  "CostLeakage",
  "Compliance",
];

export const SEVERITY_FILTERS: InsightSeverity[] = ["Critical", "Warning", "Info"];

export type InsightFilterState = {
  status: InsightStatusFilter;
  categories: InsightCategory[];
  severities: InsightSeverity[];
  page: number;
  /**
   * The finding whose detail is open, if any.
   *
   * It rides with the filters rather than being component state so an opened
   * card is part of the view a person can link to, and so closing the panel is
   * the back button.
   */
  selected: string | null;
};

export const DEFAULT_FILTERS: InsightFilterState = {
  status: "active",
  categories: [],
  severities: [],
  page: 1,
  selected: null,
};

/**
 * Expands a reader's lifecycle choice into the statuses the API understands.
 *
 * "Closed" covers both ways a finding stops being made — the condition went
 * away, or a newer run replaced it — because from the reader's side those are
 * the same thing: nobody is being asked to act on it any more.
 */
export function statusesFor(status: InsightStatusFilter): string[] {
  switch (status) {
    case "dismissed":
      return ["Dismissed"];
    case "closed":
      return ["Resolved", "Superseded"];
    default:
      return ["Active"];
  }
}

/**
 * Reads filter state out of the URL.
 *
 * The URL is the source of truth so a filtered view can be linked, reloaded and
 * navigated back to. Anything unrecognised is dropped rather than rejected: a
 * hand-edited or stale link should open the page showing something, not an
 * error, and a filter value this build has not heard of narrows nothing.
 */
export function parseFilters(params: URLSearchParams): InsightFilterState {
  const status = params.get("status");

  return {
    status: isStatusFilter(status) ? status : DEFAULT_FILTERS.status,
    categories: keepKnown(params.getAll("category"), CATEGORY_FILTERS),
    severities: keepKnown(params.getAll("severity"), SEVERITY_FILTERS),
    page: parsePage(params.get("page")),
    selected: params.get("selected"),
  };
}

/**
 * Writes filter state back to the URL, omitting anything at its default.
 *
 * A URL that spells out every default is unreadable and makes the unfiltered
 * page look filtered. Only what a person actually chose appears.
 */
export function serializeFilters(filters: InsightFilterState): URLSearchParams {
  const params = new URLSearchParams();

  if (filters.status !== DEFAULT_FILTERS.status) {
    params.set("status", filters.status);
  }

  for (const category of filters.categories) {
    params.append("category", category);
  }

  for (const severity of filters.severities) {
    params.append("severity", severity);
  }

  if (filters.page > 1) {
    params.set("page", String(filters.page));
  }

  if (filters.selected !== null && filters.selected !== "") {
    params.set("selected", filters.selected);
  }

  return params;
}

/**
 * Adds or removes one value from a multi-select filter, and returns to the first
 * page.
 *
 * Staying on page four while narrowing the result set is how someone ends up
 * looking at an empty page and concluding there is nothing to see.
 */
export function toggleFilter<T extends string>(
  filters: InsightFilterState,
  key: "categories" | "severities",
  value: T,
): InsightFilterState {
  const current = filters[key] as string[];
  const next = current.includes(value)
    ? current.filter((entry) => entry !== value)
    : [...current, value];

  return { ...filters, [key]: next, page: 1, selected: null } as InsightFilterState;
}

export function setStatusFilter(
  filters: InsightFilterState,
  status: InsightStatusFilter,
): InsightFilterState {
  return { ...filters, status, page: 1, selected: null };
}

/** Whether anything has been narrowed, for showing a "clear" affordance. */
export function hasActiveFilters(filters: InsightFilterState): boolean {
  return (
    filters.status !== DEFAULT_FILTERS.status ||
    filters.categories.length > 0 ||
    filters.severities.length > 0
  );
}

export function pageCount(total: number, pageSize: number): number {
  if (total <= 0 || pageSize <= 0) {
    return 1;
  }

  return Math.ceil(total / pageSize);
}

function isStatusFilter(value: string | null): value is InsightStatusFilter {
  return value !== null && (STATUS_FILTERS as string[]).includes(value);
}

function keepKnown<T extends string>(values: string[], known: readonly T[]): T[] {
  const seen = new Set<string>();

  return values.filter((value): value is T => {
    if (!(known as readonly string[]).includes(value) || seen.has(value)) {
      return false;
    }
    seen.add(value);

    return true;
  });
}

function parsePage(value: string | null): number {
  if (value === null) {
    return 1;
  }

  const page = Number(value);

  return Number.isInteger(page) && page > 0 ? page : 1;
}

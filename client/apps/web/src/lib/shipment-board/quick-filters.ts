import type {
  ShipmentQuickFilter,
  ShipmentQuickFilterInput,
} from "@trenova/graphql/generated/graphql";
import { createParser } from "nuqs";

export type QuickFilterToken = ShipmentQuickFilterInput;

const QUICK_FILTERS = new Set<ShipmentQuickFilter>([
  "Late",
  "Uncovered",
  "Moving",
  "DeliveringToday",
  "Reefer",
  "LowMargin",
  "DeliveryHour",
  "PickupWindow",
  "Detention",
  "ReadyToBill",
]);

/** The quick filters offered from the search field, in menu order. */
export const QUICK_FILTER_MENU = [
  { filter: "Late", label: "Late", dotClassName: "bg-danger" },
  { filter: "Uncovered", label: "Uncovered", dotClassName: "bg-warning" },
  { filter: "Moving", label: "Moving", dotClassName: "bg-info" },
  { filter: "DeliveringToday", label: "Delivering today", dotClassName: "bg-success" },
  { filter: "Reefer", label: "Reefer", dotClassName: "bg-muted-foreground" },
  { filter: "LowMargin", label: "Low margin", dotClassName: "bg-muted-foreground" },
] as const satisfies ReadonlyArray<{
  filter: ShipmentQuickFilter;
  label: string;
  dotClassName: string;
}>;

const isWholeNumber = (value: unknown, min: number, max: number) =>
  typeof value === "number" && Number.isInteger(value) && value >= min && value <= max;

function normalize(raw: unknown): QuickFilterToken | null {
  if (!raw || typeof raw !== "object") return null;
  const candidate = raw as Record<string, unknown>;
  const filter = candidate.filter as ShipmentQuickFilter;
  if (!QUICK_FILTERS.has(filter)) return null;

  if (filter === "DeliveryHour") {
    return isWholeNumber(candidate.hour, 0, 23) ? { filter, hour: candidate.hour as number } : null;
  }
  if (filter === "PickupWindow") {
    const start = candidate.windowStartMinutes;
    const end = candidate.windowEndMinutes;
    if (!isWholeNumber(start, 0, 100_000)) return null;
    if (end != null && (!isWholeNumber(end, 0, 100_000) || (end as number) <= (start as number))) {
      return null;
    }
    return end == null
      ? { filter, windowStartMinutes: start as number }
      : { filter, windowStartMinutes: start as number, windowEndMinutes: end as number };
  }
  return { filter };
}

export function parseQuickFilters(value: string): QuickFilterToken[] {
  try {
    const parsed: unknown = JSON.parse(value);
    if (!Array.isArray(parsed)) return [];
    return parsed.map(normalize).filter((token): token is QuickFilterToken => token !== null);
  } catch {
    return [];
  }
}

export function serializeQuickFilters(tokens: readonly QuickFilterToken[]): string {
  return tokens.length === 0 ? "" : JSON.stringify(tokens);
}

export const parseAsQuickFilters = createParser<QuickFilterToken[]>({
  parse: parseQuickFilters,
  serialize: serializeQuickFilters,
  eq: (a, b) => serializeQuickFilters(a) === serializeQuickFilters(b),
}).withDefault([]);

export function quickFilterKey(token: QuickFilterToken): string {
  return [token.filter, token.hour, token.windowStartMinutes, token.windowEndMinutes].join(":");
}

/** One filter per kind: picking a second hour replaces the first rather than ANDing to nothing. */
export function addQuickFilter(
  tokens: readonly QuickFilterToken[],
  token: QuickFilterToken,
): QuickFilterToken[] {
  return [...tokens.filter((existing) => existing.filter !== token.filter), token];
}

export function removeQuickFilter(
  tokens: readonly QuickFilterToken[],
  token: QuickFilterToken,
): QuickFilterToken[] {
  const key = quickFilterKey(token);
  return tokens.filter((existing) => quickFilterKey(existing) !== key);
}

type Translate = (message: string, ...args: Array<string | number>) => string;

const pad = (hour: number) => `${String(hour).padStart(2, "0")}:00`;

function windowLabel(start: number, end: number | null | undefined, t: Translate): string {
  const hours = (minutes: number) => Math.round(minutes / 60);
  if (start === 0 && end != null) return t("Pickup in < {0}h", hours(end));
  if (end == null) return t("Pickup after {0}h", hours(start));
  return t("Pickup in {0}–{1}h", hours(start), hours(end));
}

/** The words a quick filter's token reads as in the search field. */
export function quickFilterLabel(token: QuickFilterToken, t: Translate): string {
  switch (token.filter) {
    case "DeliveryHour":
      return t("Delivering {0}–{1}", pad(token.hour ?? 0), pad(((token.hour ?? 0) + 1) % 24));
    case "PickupWindow":
      return windowLabel(token.windowStartMinutes ?? 0, token.windowEndMinutes, t);
    case "Detention":
      return t("Detention accruing");
    case "ReadyToBill":
      return t("Ready to bill");
    default: {
      const menu = QUICK_FILTER_MENU.find((entry) => entry.filter === token.filter);
      return menu ? t(menu.label) : token.filter;
    }
  }
}

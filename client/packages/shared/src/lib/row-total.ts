import { formatNumber } from "@trenova/shared/i18n/format";

const ESTIMATE_FORMAT: Intl.NumberFormatOptions = {
  notation: "compact",
  maximumFractionDigits: 1,
};

/**
 * A list's row total as the list shows it: every digit when the server counted it ("9,412"),
 * compact when it is the planner's estimate of a list too large to count ("1.2M"). The caller
 * puts an estimate in a message that says it is approximate ("about {0}").
 */
export function formatRowTotal(total: number, exact: boolean): string {
  return exact ? formatNumber(total) : formatNumber(total, ESTIMATE_FORMAT);
}

/** A count a connection returned, and whether it was counted (`totalCountExact`). */
export type RowTotal = { count: number; exact: boolean };

/** The total a connection returned; no total is none counted, which is exact. */
export function rowTotalOf(
  totalCount: number | null | undefined,
  totalCountExact: boolean | null | undefined,
): RowTotal {
  return { count: totalCount ?? 0, exact: totalCount == null || totalCountExact !== false };
}

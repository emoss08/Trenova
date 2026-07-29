import { queries } from "@/lib/queries";
import { DETENTION_STATS_LIMIT } from "@/lib/queries/detention";
import type { FacilityDetentionStat } from "@trenova/shared/types/detention";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

export type DetentionWindowValue = "30" | "90" | "180";

export const DETENTION_WINDOW_OPTIONS: {
  value: DetentionWindowValue;
  label: string;
  days: number;
}[] = [
  { value: "30", label: "30d", days: 30 },
  { value: "90", label: "90d", days: 90 },
  { value: "180", label: "180d", days: 180 },
];

const SECONDS_PER_DAY = 86_400;

export function detentionWindowDays(value: DetentionWindowValue): number {
  return (
    DETENTION_WINDOW_OPTIONS.find((option) => option.value === value)?.days ??
    DETENTION_WINDOW_OPTIONS[1].days
  );
}

/** Every stop in the window, netted down to the numbers a controller argues with. */
export type DetentionRollup = {
  facilityCount: number;
  stopCount: number;
  breachCount: number;
  billed: number;
  driverPay: number;
  netMargin: number;
  waived: number;
  disputeCount: number;
  suppressedCount: number;
  /** True when the ranked query hit its ceiling, so the totals under-count. */
  truncated: boolean;
};

const EMPTY_ROLLUP: DetentionRollup = {
  facilityCount: 0,
  stopCount: 0,
  breachCount: 0,
  billed: 0,
  driverPay: 0,
  netMargin: 0,
  waived: 0,
  disputeCount: 0,
  suppressedCount: 0,
  truncated: false,
};

/**
 * Facilities are the one grouping that covers every settled stop exactly once,
 * so the page-level ledger is folded from them rather than from the customer
 * roll-up, which would double-count nothing but truncate differently.
 */
function rollupFacilities(rows: FacilityDetentionStat[] | undefined): DetentionRollup {
  if (!rows || rows.length === 0) {
    return EMPTY_ROLLUP;
  }

  const rollup = rows.reduce<DetentionRollup>(
    (totals, row) => {
      totals.stopCount += row.stopCount;
      totals.breachCount += row.breachCount;
      totals.billed += row.billedAmount;
      totals.driverPay += row.driverPayAmount;
      totals.netMargin += row.netMargin;
      totals.waived += row.waivedAmount;
      totals.disputeCount += row.disputeCount;
      totals.suppressedCount += row.suppressedCount;
      return totals;
    },
    { ...EMPTY_ROLLUP },
  );

  rollup.facilityCount = rows.length;
  rollup.truncated = rows.length >= DETENTION_STATS_LIMIT;

  return rollup;
}

export function useDetentionIntelligence(days: number) {
  const range = useMemo(() => {
    const to = Math.floor(Date.now() / 1000);
    return { from: to - days * SECONDS_PER_DAY, to };
  }, [days]);

  const facilities = useQuery(queries.detention.facilities(range.from, range.to));
  const customers = useQuery(queries.detention.customers(range.from, range.to));
  const waivers = useQuery(queries.detention.waivers(range.from, range.to));

  const rollup = useMemo(() => rollupFacilities(facilities.data), [facilities.data]);

  return { range, facilities, customers, waivers, rollup };
}

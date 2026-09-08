import type { ARDashboardKpis, ARDsoTrendPoint } from "@/lib/graphql/accounts-receivable";

/**
 * How far back the dashboard looks before calling the book empty. The trend
 * series is a generated week grid, so it is never short; what matters is
 * whether any week in it carried billing or a balance.
 */
export const AR_DASHBOARD_HISTORY_WEEKS = 52;

type OverviewSlice = Pick<
  ARDashboardKpis["overview"],
  "openInvoiceCount" | "totalOpenMinor" | "unappliedCashMinor"
>;
type TrendSlice = Pick<ARDsoTrendPoint, "billedMinor" | "arBalanceMinor">;

/**
 * Whether receivables have anything to show: an invoice open now, cash
 * waiting to be applied, or billing or a balance in any week of the trend.
 * A tenant that has never invoiced fails every one of those; a tenant whose
 * book is merely clear today still has history and keeps its dashboard.
 */
export function receivablesHaveActivity(
  overview: OverviewSlice,
  trend: readonly TrendSlice[],
): boolean {
  if (overview.openInvoiceCount > 0 || overview.totalOpenMinor !== 0) return true;
  if (overview.unappliedCashMinor !== 0) return true;
  return trend.some((point) => point.billedMinor !== 0 || point.arBalanceMinor !== 0);
}

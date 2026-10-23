import type { ARDashboardKpis } from "@/lib/graphql/accounts-receivable";

type OverviewSlice = Pick<ARDashboardKpis["overview"], "hasInvoiced" | "unappliedCashMinor">;

/**
 * Whether receivables have anything to show: an invoice has ever posted, or
 * cash is waiting to be applied. A tenant that has never invoiced fails both;
 * a tenant whose book is merely clear today has invoiced and keeps its
 * dashboard.
 */
export function receivablesHaveActivity(overview: OverviewSlice): boolean {
  return overview.hasInvoiced || overview.unappliedCashMinor !== 0;
}

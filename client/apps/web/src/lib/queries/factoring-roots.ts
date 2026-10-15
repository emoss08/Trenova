import type { QueryClient } from "@tanstack/react-query";

/**
 * Every root a factoring query caches under, kept apart from the factories
 * and their GraphQL wrappers so a page that only invalidates after a write
 * does not download the factoring surface to do it.
 */
export const FACTORING_QUERY_SCOPE = "factoring";
export const FACTORING_NOA_QUERY_SCOPE = "factoring-noa";
export const FACTORING_REMITTANCE_QUERY_SCOPE = "factoring-remittance";

export const FACTORING_INVOICE_STATE_KEY = "factoring-invoice-state-list";
export const FACTORING_SUBMISSION_KEY = "factoring-submission-list";
export const FACTORING_REMITTANCE_KEY = "factoring-remittance-list";

const FACTORING_QUERY_ROOTS = [
  FACTORING_QUERY_SCOPE,
  FACTORING_NOA_QUERY_SCOPE,
  FACTORING_REMITTANCE_QUERY_SCOPE,
  FACTORING_INVOICE_STATE_KEY,
  FACTORING_SUBMISSION_KEY,
  FACTORING_REMITTANCE_KEY,
] as const;

/**
 * Every cache a factoring change can stale: the readiness queue and its
 * summary, an invoice's factoring section, a factor's agreements, its
 * delivery settings, the notices listed on a factor or a customer, the
 * schedules sent to factors, and what the factors hold and have paid.
 */
export function invalidateFactoringQueries(queryClient: QueryClient): void {
  for (const root of FACTORING_QUERY_ROOTS) {
    void queryClient.invalidateQueries({ queryKey: [root] });
  }
}

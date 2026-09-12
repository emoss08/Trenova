import { apiService } from "@/services/api";
import type { QueryClient } from "@tanstack/react-query";

export const STATEMENT_LIST_KEY = "statement-list";
export const STATEMENT_DETAIL_KEY = "statement-detail";

/**
 * Every statement customer's open period.
 *
 * Nothing is persisted, so the server recomputes this on each read and the only
 * thing that makes it stale is the billing queue changing. Approving a shipment
 * in the Shipments view therefore has to invalidate it.
 */
export function statementListQuery() {
  return {
    queryKey: [STATEMENT_LIST_KEY] as const,
    queryFn: () => apiService.invoiceRunService.listStatements(),
  };
}

export function statementDetailQuery(customerId: string | null) {
  return {
    queryKey: [STATEMENT_DETAIL_KEY, customerId] as const,
    queryFn: () => apiService.invoiceRunService.getStatement(customerId as string),
    enabled: !!customerId,
  };
}

/** Billing a statement mints invoices and empties queue items, so all three move. */
export function invalidateStatements(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: [STATEMENT_LIST_KEY] });
  void queryClient.invalidateQueries({ queryKey: [STATEMENT_DETAIL_KEY] });
  void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
  void queryClient.invalidateQueries({ queryKey: ["invoice-list"] });
}

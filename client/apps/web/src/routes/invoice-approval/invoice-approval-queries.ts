import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import type { InvoiceAdjustmentKind } from "@trenova/graphql/generated/graphql";
import {
  fetchInvoiceAdjustmentOperationsSummary,
  fetchInvoiceApprovalDetail,
  listInvoiceAdjustmentApprovals,
} from "@/lib/graphql/invoice-adjustment";

export const INVOICE_APPROVAL_PAGE_SIZE = 20;

const INVOICE_ADJUSTMENT_KEY = "invoice-adjustment";

export type InvoiceApprovalQueueFilters = {
  query: string;
  kind: InvoiceAdjustmentKind | null;
};

/**
 * The approval queue a page at a time. Keys sit under `invoice-adjustment` so
 * every adjustment mutation's invalidation reaches them, and the search is
 * trimmed in the key so a trailing space does not refetch an identical list.
 */
export function invoiceApprovalQueueQuery({ query, kind }: InvoiceApprovalQueueFilters) {
  const search = query.trim();
  return infiniteQueryOptions({
    queryKey: [INVOICE_ADJUSTMENT_KEY, "approval-queue", search, kind] as const,
    queryFn: ({ pageParam, signal }) =>
      listInvoiceAdjustmentApprovals(
        { first: INVOICE_APPROVAL_PAGE_SIZE, after: pageParam, query: search, kind },
        { signal },
      ),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo.hasNextPage ? (lastPage.pageInfo.endCursor ?? undefined) : undefined,
  });
}

export function invoiceApprovalDetailQuery(adjustmentId: string) {
  return queryOptions({
    queryKey: [INVOICE_ADJUSTMENT_KEY, "approval-detail", adjustmentId] as const,
    queryFn: ({ signal }) => fetchInvoiceApprovalDetail(adjustmentId, { signal }),
    enabled: adjustmentId !== "",
  });
}

export function invoiceAdjustmentOperationsSummaryQuery() {
  return queryOptions({
    queryKey: [INVOICE_ADJUSTMENT_KEY, "operations-summary"] as const,
    queryFn: ({ signal }) => fetchInvoiceAdjustmentOperationsSummary({ signal }),
  });
}

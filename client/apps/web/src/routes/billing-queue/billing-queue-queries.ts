import {
  getBillingTransferRunGraphQL,
  getMyActiveBillingTransferRunGraphQL,
  listBillingTransferCandidatesGraphQL,
  listBillingTransferRunItemsGraphQL,
  type BillingTransferCandidateFilters,
  type BillingTransferItemStatus,
} from "@/lib/graphql/billing-transfer";
import { apiService } from "@/services/api";
import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { billingQueueItemSchema } from "@trenova/shared/types/billing-queue";
import { createLimitOffsetResponse } from "@trenova/shared/types/server";
import { shouldPollRun } from "./_components/bulk-transfer/bulk-billing-transfer-run";

export const BILLING_QUEUE_LIST_KEY = "billing-queue-list";
export const BILLING_QUEUE_BY_SHIPMENT_KEY = "billing-queue-by-shipment";
export const BILLING_QUEUE_FILTER_PRESETS_KEY = "billing-queue-filter-presets";
export const BILLING_TRANSFER_CANDIDATES_KEY = "billing-transfer-candidates";
export const BILLING_TRANSFER_RUN_KEY = "billing-transfer-run";
export const BILLING_TRANSFER_ACTIVE_RUN_KEY = "billing-transfer-active-run";
export const BILLING_TRANSFER_RUN_ITEMS_KEY = "billing-transfer-run-items";

const BILLING_TRANSFER_CANDIDATES_PAGE_SIZE = 50;
const BILLING_TRANSFER_RUN_ITEMS_PAGE_SIZE = 250;
const BILLING_TRANSFER_RUN_POLL_MS = 2000;

const FILTER_PRESETS_STALE_TIME_MS = 5 * 60 * 1000;

const billingQueueListSchema = createLimitOffsetResponse(billingQueueItemSchema);

export type BillingQueueListFilters = {
  status: string | null;
  billers: string[];
  billType: string | null;
  payer?: string | null;
  search: string;
  includePosted: boolean;
};

type BillingQueueFieldFilter = {
  field: string;
  operator: string;
  value: string | string[];
};

export function billingQueueFilterPresetsQuery() {
  return {
    queryKey: [BILLING_QUEUE_FILTER_PRESETS_KEY] as const,
    queryFn: () => apiService.billingQueueService.listFilterPresets(),
    staleTime: FILTER_PRESETS_STALE_TIME_MS,
  };
}

/**
 * The sidebar's item list. The key spells the filters out one by one rather than as an
 * object so a prefix invalidation on BILLING_QUEUE_LIST_KEY keeps reaching every variant.
 */
export function billingQueueListQuery({
  status,
  billers,
  billType,
  payer = null,
  search,
  includePosted,
}: BillingQueueListFilters) {
  return {
    queryKey: [
      BILLING_QUEUE_LIST_KEY,
      status,
      billers.join(","),
      billers[0],
      billType,
      search,
      includePosted,
      payer,
    ] as const,
    queryFn: async () => {
      const params = new URLSearchParams({ limit: "100" });
      const filters: BillingQueueFieldFilter[] = [];
      if (status) {
        filters.push({ field: "status", operator: "eq", value: status });
      }
      if (billers.length === 1) {
        filters.push({ field: "assignedBillerId", operator: "eq", value: billers[0] });
      } else if (billers.length > 1) {
        filters.push({ field: "assignedBillerId", operator: "in", value: billers });
      }
      if (billType) {
        filters.push({ field: "billType", operator: "eq", value: billType });
      }
      if (payer) {
        filters.push({ field: "billToCustomerId", operator: "eq", value: payer });
      }
      if (search.trim()) {
        params.set("query", search.trim());
      }
      if (includePosted) {
        params.set("includePosted", "true");
      }
      if (filters.length > 0) {
        params.set("fieldFilters", JSON.stringify(filters));
      }
      const response = await api.get(`/billing-queue/?${params.toString()}`);
      return safeParse(billingQueueListSchema, response, "BillingQueueList");
    },
  };
}

/**
 * Every queue item a shipment has produced, posted ones included: a split
 * shipment carries one per payer, and each is tracked on its own.
 */
export function billingQueueItemsByShipmentQuery(shipmentId: string) {
  return {
    queryKey: [BILLING_QUEUE_BY_SHIPMENT_KEY, shipmentId] as const,
    queryFn: async () => {
      const params = new URLSearchParams({ limit: "20", includePosted: "true" });
      params.set(
        "fieldFilters",
        JSON.stringify([{ field: "shipmentId", operator: "eq", value: shipmentId }]),
      );
      const response = await api.get(`/billing-queue/?${params.toString()}`);
      return safeParse(billingQueueListSchema, response, "BillingQueueByShipment");
    },
  };
}

/**
 * Shipments that can still move into the queue, a page at a time. The search is
 * trimmed in the key so a trailing space does not refetch an identical list.
 */
export function billingTransferCandidatesQuery({ query, status }: BillingTransferCandidateFilters) {
  const search = query.trim();
  return infiniteQueryOptions({
    queryKey: [BILLING_TRANSFER_CANDIDATES_KEY, search, status] as const,
    queryFn: ({ pageParam, signal }) =>
      listBillingTransferCandidatesGraphQL(
        {
          first: BILLING_TRANSFER_CANDIDATES_PAGE_SIZE,
          after: pageParam,
          query: search,
          status,
        },
        { signal },
      ),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo.hasNextPage ? lastPage.pageInfo.endCursor : undefined,
  });
}

/**
 * A transfer run, polled while it is still going.
 *
 * The interval is the backstop, not the delivery mechanism: the worker pushes a
 * realtime invalidation after each batch, so the bar usually moves before the
 * next poll is due. Polling stops the moment the run reaches a terminal state.
 */
export function billingTransferRunQuery(runId: string | null) {
  return queryOptions({
    queryKey: [BILLING_TRANSFER_RUN_KEY, runId] as const,
    queryFn: ({ signal }) => getBillingTransferRunGraphQL(runId as string, { signal }),
    enabled: Boolean(runId),
    refetchInterval: (query) =>
      shouldPollRun(query.state.data) ? BILLING_TRANSFER_RUN_POLL_MS : false,
  });
}

/**
 * The caller's run that has not finished, if there is one. This is what lets a
 * reopened dialog — or a second tab — pick a run back up instead of starting a
 * duplicate one.
 */
export function myActiveBillingTransferRunQuery() {
  return queryOptions({
    queryKey: [BILLING_TRANSFER_ACTIVE_RUN_KEY] as const,
    queryFn: ({ signal }) => getMyActiveBillingTransferRunGraphQL({ signal }),
    staleTime: 0,
  });
}

/** One run's per-shipment report, a page at a time. */
export function billingTransferRunItemsQuery(
  runId: string | null,
  statuses?: BillingTransferItemStatus[],
) {
  return infiniteQueryOptions({
    queryKey: [BILLING_TRANSFER_RUN_ITEMS_KEY, runId, statuses ?? null] as const,
    queryFn: ({ pageParam, signal }) =>
      listBillingTransferRunItemsGraphQL(
        {
          runId: runId as string,
          first: BILLING_TRANSFER_RUN_ITEMS_PAGE_SIZE,
          after: pageParam,
          statuses,
        },
        { signal },
      ),
    enabled: Boolean(runId),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo.hasNextPage ? lastPage.pageInfo.endCursor : undefined,
  });
}

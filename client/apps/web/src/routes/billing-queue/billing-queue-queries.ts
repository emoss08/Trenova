import {
  getBillingTransferRunGraphQL,
  getMyActiveBillingTransferRunGraphQL,
  listBillingTransferCandidatesGraphQL,
  listBillingTransferRunItemsGraphQL,
  type BillingTransferCandidateFilters,
  type BillingTransferItemStatus,
} from "@/lib/graphql/billing-transfer";
import { billingQueue } from "@/lib/queries/billing-queue";
import { apiService } from "@/services/api";
import { getStartOfDay } from "@trenova/shared/lib/date";
import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import {
  type BillingQueueItemConnection,
  type BillingQueueRow,
  listBillingQueueItemsByShipmentGraphQL,
  listBillingQueueItemsGraphQL,
} from "@/lib/graphql/billing-queue";
import type { FieldFilterInput } from "@trenova/graphql/generated/graphql";
import { dateWindowFilter, type DateWindow } from "@/lib/billing-queue-date-window";
import type { BillingQueueLane } from "@trenova/shared/types/billing-queue";
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
export const BILLING_QUEUE_LANE_PAGE_SIZE = 25;

export type BillingQueueListFilters = {
  status: string | null;
  billers: string[];
  billType: string | null;
  payer?: string | null;
  search: string;
  includePosted: boolean;
  minAmount?: string | null;
  maxAmount?: string | null;
  createdBefore?: number | null;
  shipmentCustomer?: string | null;
  delivered?: DateWindow | null;
  pickedUp?: DateWindow | null;
  queued?: DateWindow | null;
};

export function billingQueueFilterPresetsQuery() {
  return {
    queryKey: [BILLING_QUEUE_FILTER_PRESETS_KEY] as const,
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.billingQueueService.listFilterPresets({ signal }),
    staleTime: FILTER_PRESETS_STALE_TIME_MS,
  };
}

/** The queue's lane counts with the pulse of the person's own day. */
export function billingQueueStatsQuery() {
  return billingQueue.stats(getStartOfDay());
}

function fieldFiltersOf(filters: BillingQueueListFilters): FieldFilterInput[] {
  const fieldFilters: FieldFilterInput[] = [];
  if (filters.status) {
    fieldFilters.push({ field: "status", operator: "eq", value: filters.status });
  }
  if (filters.billers.length === 1) {
    fieldFilters.push({ field: "assignedBillerId", operator: "eq", value: filters.billers[0] });
  } else if (filters.billers.length > 1) {
    fieldFilters.push({ field: "assignedBillerId", operator: "in", value: filters.billers });
  }
  if (filters.billType) {
    fieldFilters.push({ field: "billType", operator: "eq", value: filters.billType });
  }
  if (filters.payer) {
    fieldFilters.push({ field: "billToCustomerId", operator: "eq", value: filters.payer });
  }
  if (filters.shipmentCustomer) {
    fieldFilters.push({
      field: "shipment.customerId",
      operator: "eq",
      value: filters.shipmentCustomer,
    });
  }
  if (filters.delivered) {
    fieldFilters.push(dateWindowFilter("shipment.actualDeliveryDate", filters.delivered));
  }
  if (filters.pickedUp) {
    fieldFilters.push(dateWindowFilter("shipment.actualShipDate", filters.pickedUp));
  }
  if (filters.queued) {
    fieldFilters.push(dateWindowFilter("createdAt", filters.queued));
  }
  return fieldFilters;
}

/**
 * One lane of the queue, a page at a time. Each lane pages on its own so a lane of
 * thousands never crowds the others out, and a collapsed lane fetches nothing. The
 * key starts with BILLING_QUEUE_LIST_KEY so every invalidation of the list reaches it.
 */
export function billingQueueLaneQuery(lane: BillingQueueLane, filters: BillingQueueListFilters) {
  const scope: BillingQueueListFilters = {
    status: filters.status,
    billers: filters.billers,
    billType: filters.billType,
    payer: filters.payer ?? null,
    search: filters.search.trim(),
    includePosted: filters.includePosted,
    minAmount: filters.minAmount ?? null,
    maxAmount: filters.maxAmount ?? null,
    createdBefore: filters.createdBefore ?? null,
    shipmentCustomer: filters.shipmentCustomer ?? null,
    delivered: filters.delivered ?? null,
    pickedUp: filters.pickedUp ?? null,
    queued: filters.queued ?? null,
  };
  return infiniteQueryOptions({
    queryKey: [BILLING_QUEUE_LIST_KEY, "lane", lane, scope] as const,
    queryFn: ({ pageParam, signal }) =>
      listBillingQueueItemsGraphQL(
        {
          first: BILLING_QUEUE_LANE_PAGE_SIZE,
          after: pageParam,
          query: scope.search,
          fieldFilters: fieldFiltersOf(scope),
          filter: {
            lanes: [lane],
            includePosted: scope.includePosted,
            minAmount: scope.minAmount ?? undefined,
            maxAmount: scope.maxAmount ?? undefined,
            createdBefore: scope.createdBefore ?? undefined,
          },
        },
        { signal },
      ),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo.hasNextPage ? lastPage.pageInfo.endCursor : undefined,
  });
}

/** The items a lane query holds, in order, across every page it has loaded. */
export function laneItems(
  data: { pages: BillingQueueItemConnection[] } | undefined,
): BillingQueueRow[] {
  return data?.pages.flatMap((page) => page.edges.map((edge) => edge.node)) ?? [];
}

/** How many items a lane holds, as its first page counted them. */
export function laneTotal(data: { pages: BillingQueueItemConnection[] } | undefined) {
  return data?.pages[0]?.totalCount ?? null;
}

/** A shipment's documents with their types, read by the Documents tab and its count. */
export function shipmentDocumentsQuery(shipmentId: string) {
  return {
    queryKey: ["documents", "shipment", shipmentId, "includeDocumentType"] as const,
    queryFn: () =>
      apiService.documentService.getByResource("shipment", shipmentId, undefined, {
        includeDocumentType: "true",
      }),
  };
}

/**
 * Every queue item a shipment has produced, posted ones included: a split
 * shipment carries one per payer, and each is tracked on its own.
 */
export function billingQueueItemsByShipmentQuery(shipmentId: string) {
  return {
    queryKey: [BILLING_QUEUE_BY_SHIPMENT_KEY, shipmentId] as const,
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      listBillingQueueItemsByShipmentGraphQL(shipmentId, { signal }),
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

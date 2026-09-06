import { apiService } from "@/services/api";
import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { billingQueueItemSchema } from "@trenova/shared/types/billing-queue";
import { createLimitOffsetResponse } from "@trenova/shared/types/server";

export const BILLING_QUEUE_LIST_KEY = "billing-queue-list";
export const BILLING_QUEUE_FILTER_PRESETS_KEY = "billing-queue-filter-presets";

const FILTER_PRESETS_STALE_TIME_MS = 5 * 60 * 1000;

const billingQueueListSchema = createLimitOffsetResponse(billingQueueItemSchema);

export type BillingQueueListFilters = {
  status: string | null;
  billers: string[];
  billType: string | null;
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

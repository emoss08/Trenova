import {
  listExceptionShipmentsGraphQL,
  listUnassignedShipmentsGraphQL,
} from "@/lib/graphql/shipment";
import { queries } from "@/lib/queries";
import type { FieldFilter } from "@trenova/shared/types/data-table";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";

const DEFAULT_LIMIT = 20;

export type ExceptionCategory = "all" | "eta-slip" | "detention" | "doc-issues";

function exceptionFilters(category: ExceptionCategory): FieldFilter[] {
  switch (category) {
    case "all":
      return [
        {
          field: "status",
          operator: "in",
          value: ["Delayed"],
        },
      ];
    case "eta-slip":
      return [{ field: "status", operator: "eq", value: "Delayed" }];
    case "detention":
      // Detention is currently surfaced via Delayed status — once the backend
      // exposes a dedicated stop-dwell metric we can refine this.
      return [{ field: "status", operator: "eq", value: "Delayed" }];
    case "doc-issues":
      return [
        {
          field: "billingTransferStatus",
          operator: "eq",
          value: "SentBackToOps",
        },
      ];
  }
}

export function useUnassignedShipments(pageSize = DEFAULT_LIMIT, enabled = true) {
  return useInfiniteQuery({
    queryKey: [...queries.shipment.listUnassigned._def, { pageSize }],
    queryFn: ({ pageParam }) =>
      listUnassignedShipmentsGraphQL({ limit: pageSize, after: pageParam }),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo?.hasNextPage ? lastPage.pageInfo.endCursor : undefined,
    staleTime: 30_000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled,
  });
}

export function useExceptionShipments(
  category: ExceptionCategory,
  limit = DEFAULT_LIMIT,
  enabled = true,
) {
  return useQuery({
    queryKey: ["shipment-list", "right-stack", "exceptions", category, { limit }],
    queryFn: () =>
      listExceptionShipmentsGraphQL({
        limit,
        fieldFilters: exceptionFilters(category),
      }),
    staleTime: 30_000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled,
  });
}

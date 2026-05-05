import { fetchData } from "@/hooks/data-table/use-data-table-query";
import type { FieldFilter } from "@/types/data-table";
import type { Shipment } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";

const SHIPMENTS_LINK = "/shipments/";
const DEFAULT_LIMIT = 20;

export type ExceptionCategory = "all" | "eta-slip" | "detention" | "doc-issues";

const UNASSIGNED_FILTERS: FieldFilter[] = [
  {
    field: "status",
    operator: "in",
    value: ["New", "PartiallyAssigned"],
  },
];

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

export function useUnassignedShipments(limit = DEFAULT_LIMIT) {
  return useQuery({
    queryKey: ["shipment-list", "right-stack", "unassigned", { limit }],
    queryFn: () =>
      fetchData<Shipment & Record<string, unknown>>(
        SHIPMENTS_LINK,
        0,
        limit,
        {
          fieldFilters: UNASSIGNED_FILTERS,
          extraSearchParams: { expandShipmentDetails: true },
        },
      ),
    staleTime: 30_000,
  });
}

export function useExceptionShipments(category: ExceptionCategory, limit = DEFAULT_LIMIT) {
  return useQuery({
    queryKey: ["shipment-list", "right-stack", "exceptions", category, { limit }],
    queryFn: () =>
      fetchData<Shipment & Record<string, unknown>>(
        SHIPMENTS_LINK,
        0,
        limit,
        {
          fieldFilters: exceptionFilters(category),
          extraSearchParams: { expandShipmentDetails: true },
        },
      ),
    staleTime: 30_000,
  });
}

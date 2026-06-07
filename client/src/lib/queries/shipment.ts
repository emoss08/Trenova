import { apiService } from "@/services/api";
import type { PaginationInfo } from "@/types/server";
import type { Shipment } from "@/types/shipment";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const shipment = createQueryKeys("shipment", {
  uiPolicy: () => ({
    queryKey: ["ui-policy"],
    queryFn: async () => apiService.shipmentService.getUIPolicy(),
  }),
  get: (shipmentId: Shipment["id"], params?: Record<string, string>) => ({
    queryKey: ["get", shipmentId, params],
    queryFn: async () => apiService.shipmentService.get(shipmentId, params),
  }),
  billingReadiness: (shipmentId: Shipment["id"]) => ({
    queryKey: ["billing-readiness", shipmentId],
    queryFn: async () => apiService.shipmentService.getBillingReadiness(shipmentId),
  }),
  listUnassigned: (req: { limit: number; after?: string | null }) => ({
    queryKey: ["list-unassigned", req],
    queryFn: async () => apiService.shipmentService.listUnassigned(req),
  }),
  listComments: (req: PaginationInfo & { shipmentId: Shipment["id"] }) => ({
    queryKey: ["comments", req.shipmentId, req],
    queryFn: async () => apiService.shipmentService.getComments(req),
  }),
});

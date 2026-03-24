import { apiService } from "@/services/api";
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
});

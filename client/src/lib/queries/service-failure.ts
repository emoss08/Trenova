import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const serviceFailure = createQueryKeys("serviceFailure", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => apiService.serviceFailureService.getById(id),
  }),
  listByShipment: (shipmentId: string) => ({
    queryKey: ["list-by-shipment", shipmentId],
    queryFn: async () =>
      apiService.serviceFailureService.listByShipment(
        shipmentId,
        "limit=100",
      ),
  }),
});

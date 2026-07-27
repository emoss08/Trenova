import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const detention = createQueryKeys("detention", {
  desk: () => ({
    queryKey: ["desk"],
    queryFn: async () => apiService.detentionService.desk(),
  }),
  byShipment: (shipmentId: string) => ({
    queryKey: ["byShipment", shipmentId],
    queryFn: async () => apiService.detentionService.byShipment(shipmentId),
  }),
  occurrence: (id: string) => ({
    queryKey: ["occurrence", id],
    queryFn: async () => apiService.detentionService.getOccurrence(id),
  }),
  disputePacket: (id: string) => ({
    queryKey: ["disputePacket", id],
    queryFn: async () => apiService.detentionService.disputePacket(id),
  }),
});

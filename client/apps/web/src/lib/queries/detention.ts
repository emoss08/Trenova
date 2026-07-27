import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const detention = createQueryKeys("detention", {
  desk: () => ({
    queryKey: ["desk"],
    queryFn: async () => apiService.detentionService.desk(),
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

import { apiService } from "@/services/api";
import type { CollectionWorklistParams } from "@/types/collection";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const collection = createQueryKeys("collection", {
  worklist: (params: CollectionWorklistParams) => ({
    queryKey: ["worklist", params],
    queryFn: async () => apiService.collectionService.worklist(params),
  }),
  account: (customerId: string) => ({
    queryKey: ["account", customerId],
    queryFn: async () => apiService.collectionService.getAccount(customerId),
  }),
});

import { apiService } from "@/services/api";
import type { CollectionWorklistParams } from "@/types/collection";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const collection = createQueryKeys("collection", {
  worklist: (params: CollectionWorklistParams) => ({
    queryKey: ["worklist", params],
    queryFn: async ({ signal }) => apiService.collectionService.worklist(params, { signal }),
  }),
  account: (customerId: string) => ({
    queryKey: ["account", customerId],
    queryFn: async ({ signal }) =>
      apiService.collectionService.getAccount(customerId, undefined, { signal }),
  }),
});

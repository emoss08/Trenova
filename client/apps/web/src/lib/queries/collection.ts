import { collectionService } from "@/services/collection";
import type { CollectionWorklistParams } from "@/types/collection";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const collection = createQueryKeys("collection", {
  worklist: (params: CollectionWorklistParams) => ({
    queryKey: ["worklist", params],
    queryFn: async () => collectionService.worklist(params),
  }),
  account: (customerId: string) => ({
    queryKey: ["account", customerId],
    queryFn: async () => collectionService.getAccount(customerId),
  }),
});

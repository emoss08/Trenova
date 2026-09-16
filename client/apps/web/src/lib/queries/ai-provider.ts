import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const aiProvider = createQueryKeys("aiProvider", {
  list: () => ({
    queryKey: ["ai-provider-list"],
    queryFn: () => apiService.aiProviderService.list(),
  }),
  detail: (id: string) => ({
    queryKey: ["ai-provider", id],
    queryFn: () => apiService.aiProviderService.get(id),
  }),
  catalog: () => ({
    queryKey: ["ai-provider-catalog"],
    queryFn: () => apiService.aiProviderService.catalog(),
  }),
});

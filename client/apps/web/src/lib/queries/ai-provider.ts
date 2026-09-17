import { fetchAIProvider, fetchAIProviders } from "@/lib/graphql/ai-provider";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const aiProvider = createQueryKeys("aiProvider", {
  list: () => ({
    queryKey: ["ai-provider-list"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIProviders({ signal }),
  }),
  detail: (id: string) => ({
    queryKey: ["ai-provider", id],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIProvider(id, { signal }),
  }),
  catalog: () => ({
    queryKey: ["ai-provider-catalog"],
    queryFn: () => apiService.aiProviderService.catalog(),
  }),
});

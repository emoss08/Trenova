import {
  fetchAIProvider,
  fetchAIProviderLimits,
  fetchAIProviderModels,
  fetchAIProviders,
  fetchAIProviderUsageDaily,
  fetchAIRoutePreview,
  type AIProviderEndpoint,
  type AIProviderRoutingDraft,
} from "@/lib/graphql/ai-provider";
import { fetchAIUsageSummary } from "@/lib/graphql/ai-usage";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const aiProvider = createQueryKeys("aiProvider", {
  list: () => ({
    queryKey: ["ai-provider-list"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIProviders({ signal }),
  }),
  limits: () => ({
    queryKey: ["ai-provider-limits"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIProviderLimits({ signal }),
  }),
  detail: (id: string) => ({
    queryKey: ["ai-provider", id],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIProvider(id, { signal }),
  }),
  catalog: () => ({
    queryKey: ["ai-provider-catalog"],
    queryFn: () => apiService.aiProviderService.catalog(),
  }),
  models: (endpoint: AIProviderEndpoint) => ({
    queryKey: [endpoint],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIProviderModels(endpoint, { signal }),
  }),
  routePreview: (draft: AIProviderRoutingDraft) => ({
    queryKey: [draft],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIRoutePreview(draft, { signal }),
  }),
  daily: (providerId: string, days: number, timezone: string) => ({
    queryKey: [providerId, days, timezone],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAIProviderUsageDaily(providerId, days, timezone, { signal }),
  }),
  usage: (days: number) => ({
    queryKey: ["ai-usage-summary", days],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIUsageSummary(days, { signal }),
  }),
});

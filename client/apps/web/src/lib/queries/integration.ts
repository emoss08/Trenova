import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const integration = createQueryKeys("integration", {
  catalog: () => ({
    queryKey: ["catalog"],
    queryFn: ({ signal }) => apiService.integrationService.getCatalog({ signal }),
  }),
  config: (type: string) => ({
    queryKey: ["integration-config", type],
    queryFn: ({ signal }) => apiService.integrationService.getConfig(type, { signal }),
  }),
  runtimeConfig: (type: string) => ({
    queryKey: ["runtime-config", type],
    queryFn: ({ signal }) => apiService.integrationService.getRuntimeConfig(type, { signal }),
  }),
  apiKeys: () => ({
    queryKey: ["api-keys"],
    queryFn: () => apiService.apiKeyService.list(),
  }),
  apiKey: (id: string) => ({
    queryKey: ["api-key", id],
    queryFn: () => apiService.apiKeyService.get(id),
  }),
  samsaraWorkerSyncReadiness: () => ({
    queryKey: ["samsara-worker-sync-readiness"],
    queryFn: ({ signal }) =>
      apiService.integrationService.getSamsaraWorkerSyncReadiness({ signal }),
  }),
  samsaraWorkerSyncDrift: () => ({
    queryKey: ["samsara-worker-sync-drift"],
    queryFn: ({ signal }) => apiService.integrationService.getSamsaraWorkerSyncDrift({ signal }),
  }),
  samsaraWorkerSyncStatus: (workflowId: string, runId?: string) => ({
    queryKey: ["samsara-worker-sync-status", workflowId, runId ?? ""],
    queryFn: ({ signal }) =>
      apiService.integrationService.getSamsaraWorkerSyncStatus(workflowId, runId, { signal }),
  }),
});

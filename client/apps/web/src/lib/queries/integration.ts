import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const integration = createQueryKeys("integration", {
  catalog: () => ({
    queryKey: ["catalog"],
    queryFn: () => apiService.integrationService.getCatalog(),
  }),
  config: (type: string) => ({
    queryKey: ["integration-config", type],
    queryFn: () => apiService.integrationService.getConfig(type),
  }),
  runtimeConfig: (type: string) => ({
    queryKey: ["runtime-config", type],
    queryFn: () => apiService.integrationService.getRuntimeConfig(type),
  }),
  apiKey: (id: string) => ({
    queryKey: ["api-key", id],
    queryFn: ({ signal }) => apiService.apiKeyService.get(id, { signal }),
  }),
  samsaraWorkerSyncReadiness: () => ({
    queryKey: ["samsara-worker-sync-readiness"],
    queryFn: () => apiService.integrationService.getSamsaraWorkerSyncReadiness(),
  }),
  samsaraWorkerSyncDrift: () => ({
    queryKey: ["samsara-worker-sync-drift"],
    queryFn: () => apiService.integrationService.getSamsaraWorkerSyncDrift(),
  }),
  samsaraWorkerSyncStatus: (workflowId: string, runId?: string) => ({
    queryKey: ["samsara-worker-sync-status", workflowId, runId ?? ""],
    queryFn: () => apiService.integrationService.getSamsaraWorkerSyncStatus(workflowId, runId),
  }),
});

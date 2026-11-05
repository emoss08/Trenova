import { integrationService } from "@/services/integration";
import { apiKeyService } from "@/services/api-key";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const integration = createQueryKeys("integration", {
  catalog: () => ({
    queryKey: ["catalog"],
    queryFn: () => integrationService.getCatalog(),
  }),
  config: (type: string) => ({
    queryKey: ["integration-config", type],
    queryFn: () => integrationService.getConfig(type),
  }),
  runtimeConfig: (type: string) => ({
    queryKey: ["runtime-config", type],
    queryFn: () => integrationService.getRuntimeConfig(type),
  }),
  apiKeys: () => ({
    queryKey: ["api-keys"],
    queryFn: () => apiKeyService.list(),
  }),
  apiKey: (id: string) => ({
    queryKey: ["api-key", id],
    queryFn: () => apiKeyService.get(id),
  }),
  samsaraWorkerSyncReadiness: () => ({
    queryKey: ["samsara-worker-sync-readiness"],
    queryFn: () => integrationService.getSamsaraWorkerSyncReadiness(),
  }),
  samsaraWorkerSyncDrift: () => ({
    queryKey: ["samsara-worker-sync-drift"],
    queryFn: () => integrationService.getSamsaraWorkerSyncDrift(),
  }),
  samsaraWorkerSyncStatus: (workflowId: string, runId?: string) => ({
    queryKey: ["samsara-worker-sync-status", workflowId, runId ?? ""],
    queryFn: () => integrationService.getSamsaraWorkerSyncStatus(workflowId, runId),
  }),
});

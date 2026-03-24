import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  integrationCatalogResponseSchema,
  integrationConfigResponseSchema,
  updateIntegrationConfigRequestSchema,
  type UpdateIntegrationConfigRequest,
} from "@/types/integration";
import {
  repairWorkerSyncDriftRequestSchema,
  repairWorkerSyncDriftResponseSchema,
  syncWorkflowStartResponseSchema,
  syncWorkflowStatusResponseSchema,
  testSamsaraConnectionResponse,
  workerSyncDriftResponseSchema,
  workerSyncReadinessResponseSchema,
} from "@/types/samsara";

export class IntegrationService {
  public async getCatalog() {
    const response = await api.get("/integrations/catalog/");
    return safeParse(integrationCatalogResponseSchema, response, "Integration Catalog");
  }

  public async getConfig(type: string) {
    const response = await api.get(`/integrations/${type}/config/`);
    return safeParse(integrationConfigResponseSchema, response, `${type} Config`);
  }

  public async updateConfig(type: string, payload: UpdateIntegrationConfigRequest) {
    const request = updateIntegrationConfigRequestSchema.parse(payload);
    const response = await api.put(`/integrations/${type}/config/`, request);
    return safeParse(integrationConfigResponseSchema, response, `${type} Config`);
  }

  public async testConnection(type: string) {
    const response = await api.post(`/integrations/${type}/test-connection/`);
    return safeParse(testSamsaraConnectionResponse, response, `${type} Connection Test`);
  }

  public async getRuntimeConfig(type: string) {
    return api.get<Record<string, string>>(`/integrations/${type}/runtime-config/`);
  }

  public async getSamsaraWorkerSyncReadiness() {
    const response = await api.get("/integrations/samsara/workers/sync/readiness/");
    return safeParse(workerSyncReadinessResponseSchema, response, "Worker Sync Readiness");
  }

  public async getSamsaraWorkerSyncDrift() {
    const response = await api.get("/integrations/samsara/workers/sync/drift/");
    return safeParse(workerSyncDriftResponseSchema, response, "Worker Sync Drift");
  }

  public async detectSamsaraWorkerSyncDrift() {
    const response = await api.post("/integrations/samsara/workers/sync/drift/detect/");
    return safeParse(workerSyncDriftResponseSchema, response, "Worker Sync Drift");
  }

  public async repairSamsaraWorkerSyncDrift(workerIds: string[] = []) {
    const payload = repairWorkerSyncDriftRequestSchema.parse({ workerIds });
    const response = await api.post("/integrations/samsara/workers/sync/drift/repair/", payload);
    return safeParse(repairWorkerSyncDriftResponseSchema, response, "Worker Sync Drift Repair");
  }

  public async startSamsaraWorkerSync() {
    const response = await api.post("/integrations/samsara/workers/sync/");
    return safeParse(syncWorkflowStartResponseSchema, response, "Sync Workflow Start");
  }

  public async getSamsaraWorkerSyncStatus(workflowId: string, runId?: string) {
    const endpoint = runId
      ? `/integrations/samsara/workers/sync/${workflowId}/?runId=${encodeURIComponent(runId)}`
      : `/integrations/samsara/workers/sync/${workflowId}/`;

    const response = await api.get(endpoint);
    return safeParse(syncWorkflowStatusResponseSchema, response, "Sync Workflow Status");
  }
}

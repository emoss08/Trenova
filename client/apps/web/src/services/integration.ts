import { translate } from "@trenova/shared/i18n/runtime";
import {
  DetectSamsaraWorkerSyncDriftDocument,
  IntegrationCatalogDocument,
  IntegrationConfigDocument,
  IntegrationRuntimeConfigDocument,
  RepairSamsaraWorkerSyncDriftDocument,
  SamsaraWorkerSyncDriftDocument,
  SamsaraWorkerSyncReadinessDocument,
  SamsaraWorkerSyncRunStatusDocument,
  StartSamsaraWorkerSyncDocument,
  TestIntegrationConnectionDocument,
  UpdateIntegrationConfigDocument,
  type IntegrationType,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  integrationCatalogResponseSchema,
  integrationConfigResponseSchema,
  integrationRuntimeConfigResponseSchema,
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

type RequestOptions = { signal?: AbortSignal };

/** The integration types the server knows are its enum; a route parameter is only a string. */
function integrationType(type: string): IntegrationType {
  return type as IntegrationType;
}

export class IntegrationService {
  public async getCatalog(options: RequestOptions = {}) {
    const data = await requestGraphQL({
      document: IntegrationCatalogDocument,
      operationName: "IntegrationCatalog",
      signal: options.signal,
    });
    return safeParse(
      integrationCatalogResponseSchema,
      data.integrationCatalog,
      "Integration Catalog",
    );
  }

  public async getConfig(type: string, options: RequestOptions = {}) {
    const data = await requestGraphQL({
      document: IntegrationConfigDocument,
      operationName: "IntegrationConfig",
      variables: { type: integrationType(type) },
      signal: options.signal,
    });
    return safeParse(
      integrationConfigResponseSchema,
      data.integrationConfig,
      translate("{0} Config", type),
    );
  }

  public async updateConfig(type: string, payload: UpdateIntegrationConfigRequest) {
    const request = updateIntegrationConfigRequestSchema.parse(payload);
    const data = await requestGraphQL({
      document: UpdateIntegrationConfigDocument,
      operationName: "UpdateIntegrationConfig",
      variables: {
        input: {
          type: integrationType(type),
          enabled: request.enabled,
          configuration: Object.entries(request.configuration).map(([key, value]) => ({
            key,
            value,
          })),
        },
      },
    });
    return safeParse(
      integrationConfigResponseSchema,
      data.updateIntegrationConfig,
      translate("{0} Config", type),
    );
  }

  public async testConnection(type: string) {
    const data = await requestGraphQL({
      document: TestIntegrationConnectionDocument,
      operationName: "TestIntegrationConnection",
      variables: { type: integrationType(type) },
    });
    return safeParse(
      testSamsaraConnectionResponse,
      data.testIntegrationConnection,
      translate("{0} Connection Test", type),
    );
  }

  public async getRuntimeConfig(type: string, options: RequestOptions = {}) {
    const data = await requestGraphQL({
      document: IntegrationRuntimeConfigDocument,
      operationName: "IntegrationRuntimeConfig",
      variables: { type: integrationType(type) },
      signal: options.signal,
    });
    return safeParse(
      integrationRuntimeConfigResponseSchema,
      data.integrationRuntimeConfig,
      translate("{0} Runtime Config", type),
    );
  }

  public async getSamsaraWorkerSyncReadiness(options: RequestOptions = {}) {
    const data = await requestGraphQL({
      document: SamsaraWorkerSyncReadinessDocument,
      operationName: "SamsaraWorkerSyncReadiness",
      signal: options.signal,
    });
    return safeParse(
      workerSyncReadinessResponseSchema,
      data.samsaraWorkerSyncReadiness,
      "Worker Sync Readiness",
    );
  }

  public async getSamsaraWorkerSyncDrift(options: RequestOptions = {}) {
    const data = await requestGraphQL({
      document: SamsaraWorkerSyncDriftDocument,
      operationName: "SamsaraWorkerSyncDrift",
      signal: options.signal,
    });
    return safeParse(
      workerSyncDriftResponseSchema,
      data.samsaraWorkerSyncDrift,
      "Worker Sync Drift",
    );
  }

  public async detectSamsaraWorkerSyncDrift() {
    const data = await requestGraphQL({
      document: DetectSamsaraWorkerSyncDriftDocument,
      operationName: "DetectSamsaraWorkerSyncDrift",
    });
    return safeParse(
      workerSyncDriftResponseSchema,
      data.detectSamsaraWorkerSyncDrift,
      "Worker Sync Drift",
    );
  }

  public async repairSamsaraWorkerSyncDrift(workerIds: string[] = []) {
    const payload = repairWorkerSyncDriftRequestSchema.parse({ workerIds });
    const data = await requestGraphQL({
      document: RepairSamsaraWorkerSyncDriftDocument,
      operationName: "RepairSamsaraWorkerSyncDrift",
      variables: { workerIds: payload.workerIds },
    });
    return safeParse(
      repairWorkerSyncDriftResponseSchema,
      data.repairSamsaraWorkerSyncDrift,
      "Worker Sync Drift Repair",
    );
  }

  public async startSamsaraWorkerSync() {
    const data = await requestGraphQL({
      document: StartSamsaraWorkerSyncDocument,
      operationName: "StartSamsaraWorkerSync",
    });
    return safeParse(
      syncWorkflowStartResponseSchema,
      data.startSamsaraWorkerSync,
      "Sync Workflow Start",
    );
  }

  public async getSamsaraWorkerSyncStatus(
    workflowId: string,
    runId?: string,
    options: RequestOptions = {},
  ) {
    const data = await requestGraphQL({
      document: SamsaraWorkerSyncRunStatusDocument,
      operationName: "SamsaraWorkerSyncRunStatus",
      variables: { workflowId, runId: runId || undefined },
      signal: options.signal,
    });
    return safeParse(
      syncWorkflowStatusResponseSchema,
      data.samsaraWorkerSyncRunStatus,
      "Sync Workflow Status",
    );
  }
}

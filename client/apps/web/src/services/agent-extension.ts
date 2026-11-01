import {
  AgentExtensionCatalogDocument,
  AgentExtensionConfigDocument,
  AgentPortalRunDocument,
  AgentPortalRunLiveDocument,
  AgentPortalRunRecordingDocument,
  AgentPortalRunsDocument,
  AgentPortalSitesDocument,
  AgentWebWatchFindingsDocument,
  AgentWebWatchesDocument,
  CreateAgentPortalSiteDocument,
  DeleteAgentPortalSiteDocument,
  ProvideAgentPortalSignInCodeDocument,
  StopAgentWebWatchDocument,
  TestAgentExtensionDocument,
  UpdateAgentExtensionConfigDocument,
  UpdateAgentPortalSiteDocument,
} from "@trenova/graphql/generated/graphql";
import { translate } from "@trenova/shared/i18n/runtime";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  agentExtensionCatalogResponseSchema,
  agentExtensionConfigResponseSchema,
  agentExtensionTestResponseSchema,
  portalLiveViewSchema,
  portalRecordingSchema,
  portalRunSchema,
  portalRunsResponseSchema,
  webWatchFindingsResponseSchema,
  webWatchSchema,
  webWatchesResponseSchema,
  portalSiteSchema,
  portalSitesResponseSchema,
  savePortalSiteRequestSchema,
  updateAgentExtensionRequestSchema,
  type SavePortalSiteRequest,
  type UpdateAgentExtensionRequest,
} from "@/types/agent-extension";

export type ListPortalRunsParams = {
  siteId?: string;
  before?: number;
  limit?: number;
};

export class AgentExtensionService {
  public async getCatalog(options?: { signal?: AbortSignal }) {
    const data = await requestGraphQL({
      document: AgentExtensionCatalogDocument,
      operationName: "AgentExtensionCatalog",
      signal: options?.signal,
    });
    return safeParse(
      agentExtensionCatalogResponseSchema,
      data.agentExtensionCatalog,
      "Extension Catalog",
    );
  }

  public async getConfig(type: string, options?: { signal?: AbortSignal }) {
    const data = await requestGraphQL({
      document: AgentExtensionConfigDocument,
      operationName: "AgentExtensionConfig",
      variables: { type },
      signal: options?.signal,
    });
    return safeParse(
      agentExtensionConfigResponseSchema,
      data.agentExtensionConfig,
      translate("{0} Extension Settings", type),
    );
  }

  public async updateConfig(type: string, payload: UpdateAgentExtensionRequest) {
    const request = updateAgentExtensionRequestSchema.parse(payload);
    const data = await requestGraphQL({
      document: UpdateAgentExtensionConfigDocument,
      operationName: "UpdateAgentExtensionConfig",
      variables: {
        type,
        input: {
          enabled: request.enabled,
          availability: request.availability,
          configuration: Object.entries(request.configuration).map(([key, value]) => ({
            key,
            value,
          })),
          version: request.version,
        },
      },
    });
    return safeParse(
      agentExtensionConfigResponseSchema,
      data.updateAgentExtensionConfig,
      translate("{0} Extension Settings", type),
    );
  }

  public async test(type: string) {
    const data = await requestGraphQL({
      document: TestAgentExtensionDocument,
      operationName: "TestAgentExtension",
      variables: { type },
    });
    return safeParse(
      agentExtensionTestResponseSchema,
      data.testAgentExtension,
      translate("{0} Extension Test", type),
    );
  }

  public async listPortalSites(options?: { signal?: AbortSignal }) {
    const data = await requestGraphQL({
      document: AgentPortalSitesDocument,
      operationName: "AgentPortalSites",
      signal: options?.signal,
    });
    return safeParse(
      portalSitesResponseSchema,
      { items: data.agentPortalSites },
      translate("Portal sites"),
    );
  }

  public async createPortalSite(payload: SavePortalSiteRequest) {
    const input = savePortalSiteRequestSchema.parse(payload);
    const data = await requestGraphQL({
      document: CreateAgentPortalSiteDocument,
      operationName: "CreateAgentPortalSite",
      variables: { input },
    });
    return safeParse(portalSiteSchema, data.createAgentPortalSite, translate("Portal site"));
  }

  public async updatePortalSite(id: string, payload: SavePortalSiteRequest) {
    const input = savePortalSiteRequestSchema.parse(payload);
    const data = await requestGraphQL({
      document: UpdateAgentPortalSiteDocument,
      operationName: "UpdateAgentPortalSite",
      variables: { id, input },
    });
    return safeParse(portalSiteSchema, data.updateAgentPortalSite, translate("Portal site"));
  }

  public async deletePortalSite(id: string) {
    await requestGraphQL({
      document: DeleteAgentPortalSiteDocument,
      operationName: "DeleteAgentPortalSite",
      variables: { id },
    });
  }

  public async listPortalRuns(params: ListPortalRunsParams, options?: { signal?: AbortSignal }) {
    const data = await requestGraphQL({
      document: AgentPortalRunsDocument,
      operationName: "AgentPortalRuns",
      variables: {
        siteId: params.siteId || null,
        before: params.before || null,
        limit: params.limit || null,
      },
      signal: options?.signal,
    });
    return safeParse(portalRunsResponseSchema, data.agentPortalRuns, translate("Portal tasks"));
  }

  /**
   * Continues a portal task waiting on a sign-in code with the code the person was sent. The
   * code goes only to the browser, never into a conversation.
   */
  public async sendPortalSignInCode(runId: string, code: string) {
    return this.provideSignInCode(runId, { code, signedIn: false });
  }

  /**
   * Carries on a portal task waiting on a sign-in code after the person signed in themselves in
   * the live browser, so there is no code to send.
   */
  public async confirmPortalSignIn(runId: string) {
    return this.provideSignInCode(runId, { code: "", signedIn: true });
  }

  private async provideSignInCode(runId: string, input: { code: string; signedIn: boolean }) {
    const data = await requestGraphQL({
      document: ProvideAgentPortalSignInCodeDocument,
      operationName: "ProvideAgentPortalSignInCode",
      variables: { id: runId, input },
    });
    return safeParse(portalRunSchema, data.provideAgentPortalSignInCode, translate("Portal task"));
  }

  /** One portal task, for the person it was asked for: the Desk's card follows it as it runs. */
  public async portalRun(runId: string, options?: { signal?: AbortSignal }) {
    const data = await requestGraphQL({
      document: AgentPortalRunDocument,
      operationName: "AgentPortalRun",
      variables: { id: runId },
      signal: options?.signal,
    });
    return safeParse(portalRunSchema, data.agentPortalRun, translate("Portal task"));
  }

  /**
   * The address of the open browser a portal task runs in. It is signed in to the portal, so it
   * is read when the person asks to open it and never kept.
   */
  public async portalRunLive(runId: string) {
    const data = await requestGraphQL({
      document: AgentPortalRunLiveDocument,
      operationName: "AgentPortalRunLive",
      variables: { id: runId },
    });
    return safeParse(portalLiveViewSchema, data.agentPortalRunLive, translate("Live browser"));
  }

  public async listWebWatches(
    params: { status?: string; before?: number; limit?: number },
    options?: { signal?: AbortSignal },
  ) {
    const data = await requestGraphQL({
      document: AgentWebWatchesDocument,
      operationName: "AgentWebWatches",
      variables: {
        status: params.status || null,
        before: params.before || null,
        limit: params.limit || null,
      },
      signal: options?.signal,
    });
    return safeParse(webWatchesResponseSchema, data.agentWebWatches, translate("Web watches"));
  }

  public async webWatchFindings(id: string, options?: { signal?: AbortSignal }) {
    const data = await requestGraphQL({
      document: AgentWebWatchFindingsDocument,
      operationName: "AgentWebWatchFindings",
      variables: { id },
      signal: options?.signal,
    });
    return safeParse(
      webWatchFindingsResponseSchema,
      { items: data.agentWebWatchFindings },
      translate("Web watch findings"),
    );
  }

  /** Stops a web watch, so the vendor searches and bills for it no more. */
  public async stopWebWatch(id: string) {
    const data = await requestGraphQL({
      document: StopAgentWebWatchDocument,
      operationName: "StopAgentWebWatch",
      variables: { id },
    });
    return safeParse(webWatchSchema, data.stopAgentWebWatch, translate("Web watch"));
  }

  public async portalRunRecording(id: string) {
    const data = await requestGraphQL({
      document: AgentPortalRunRecordingDocument,
      operationName: "AgentPortalRunRecording",
      variables: { id },
    });
    return safeParse(
      portalRecordingSchema,
      data.agentPortalRunRecording,
      translate("Portal task recording"),
    );
  }
}

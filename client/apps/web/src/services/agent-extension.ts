import { translate } from "@trenova/shared/i18n/runtime";
import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  agentExtensionCatalogResponseSchema,
  agentExtensionConfigResponseSchema,
  agentExtensionTestResponseSchema,
  analysisRunsResponseSchema,
  portalRecordingSchema,
  portalRunSchema,
  portalRunsResponseSchema,
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
  public async getCatalog() {
    const response = await api.get("/agent-extensions/catalog/");
    return safeParse(agentExtensionCatalogResponseSchema, response, "Extension Catalog");
  }

  public async getConfig(type: string) {
    const response = await api.get(`/agent-extensions/${encodeURIComponent(type)}/config/`);
    return safeParse(
      agentExtensionConfigResponseSchema,
      response,
      translate("{0} Extension Settings", type),
    );
  }

  public async updateConfig(type: string, payload: UpdateAgentExtensionRequest) {
    const request = updateAgentExtensionRequestSchema.parse(payload);
    const response = await api.put(
      `/agent-extensions/${encodeURIComponent(type)}/config/`,
      request,
    );
    return safeParse(
      agentExtensionConfigResponseSchema,
      response,
      translate("{0} Extension Settings", type),
    );
  }

  public async test(type: string) {
    const response = await api.post(`/agent-extensions/${encodeURIComponent(type)}/test/`);
    return safeParse(
      agentExtensionTestResponseSchema,
      response,
      translate("{0} Extension Test", type),
    );
  }

  public async listPortalSites(options?: { signal?: AbortSignal }) {
    const response = await api.get("/agent-portals/sites/", { signal: options?.signal });
    return safeParse(portalSitesResponseSchema, response, translate("Portal sites"));
  }

  public async createPortalSite(payload: SavePortalSiteRequest) {
    const request = savePortalSiteRequestSchema.parse(payload);
    const response = await api.post("/agent-portals/sites/", request);
    return safeParse(portalSiteSchema, response, translate("Portal site"));
  }

  public async updatePortalSite(id: string, payload: SavePortalSiteRequest) {
    const request = savePortalSiteRequestSchema.parse(payload);
    const response = await api.put(`/agent-portals/sites/${encodeURIComponent(id)}/`, request);
    return safeParse(portalSiteSchema, response, translate("Portal site"));
  }

  public async deletePortalSite(id: string) {
    await api.delete(`/agent-portals/sites/${encodeURIComponent(id)}/`);
  }

  public async listPortalRuns(params: ListPortalRunsParams, options?: { signal?: AbortSignal }) {
    const query = new URLSearchParams();
    if (params.siteId) query.set("siteId", params.siteId);
    if (params.before) query.set("before", String(params.before));
    if (params.limit) query.set("limit", String(params.limit));
    const suffix = query.size > 0 ? `?${query.toString()}` : "";
    const response = await api.get(`/agent-portals/runs/${suffix}`, { signal: options?.signal });
    return safeParse(portalRunsResponseSchema, response, translate("Portal tasks"));
  }

  public async listAnalysisRuns(
    params: { before?: number; limit?: number },
    options?: { signal?: AbortSignal },
  ) {
    const query = new URLSearchParams();
    if (params.before) query.set("before", String(params.before));
    if (params.limit) query.set("limit", String(params.limit));
    const suffix = query.size > 0 ? `?${query.toString()}` : "";
    const response = await api.get(`/agent-analysis/runs/${suffix}`, { signal: options?.signal });
    return safeParse(analysisRunsResponseSchema, response, translate("Analyses"));
  }

  /** One of an analysis's charts, as the image it is, under the session's own credentials. */
  public async analysisChart(runId: string, position: number, options?: { signal?: AbortSignal }) {
    const file = await api.download(
      `/agent-analysis/runs/${encodeURIComponent(runId)}/charts/${position}/`,
      `chart-${position}.png`,
      { signal: options?.signal },
    );
    return file.blob;
  }

  /**
   * Continues a portal task waiting on a sign-in code with the code the person was sent. The
   * code goes only to the browser, never into a conversation.
   */
  public async sendPortalSignInCode(runId: string, code: string) {
    const response = await api.post(
      `/agent-portals/runs/${encodeURIComponent(runId)}/sign-in-code/`,
      { code },
    );
    return safeParse(portalRunSchema, response, translate("Portal task"));
  }

  public async portalRunRecording(id: string) {
    const response = await api.get(`/agent-portals/runs/${encodeURIComponent(id)}/recording/`);
    return safeParse(portalRecordingSchema, response, translate("Portal task recording"));
  }
}

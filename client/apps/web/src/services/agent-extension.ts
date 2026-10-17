import { translate } from "@trenova/shared/i18n/runtime";
import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  agentExtensionCatalogResponseSchema,
  agentExtensionConfigResponseSchema,
  agentExtensionTestResponseSchema,
  portalRecordingSchema,
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

  public async portalRunRecording(id: string) {
    const response = await api.get(`/agent-portals/runs/${encodeURIComponent(id)}/recording/`);
    return safeParse(portalRecordingSchema, response, translate("Portal task recording"));
  }
}

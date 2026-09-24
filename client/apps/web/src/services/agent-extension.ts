import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  agentExtensionCatalogResponseSchema,
  agentExtensionConfigResponseSchema,
  agentExtensionTestResponseSchema,
  updateAgentExtensionRequestSchema,
  type UpdateAgentExtensionRequest,
} from "@/types/agent-extension";

export class AgentExtensionService {
  public async getCatalog() {
    const response = await api.get("/agent-extensions/catalog/");
    return safeParse(agentExtensionCatalogResponseSchema, response, "Extension Catalog");
  }

  public async getConfig(type: string) {
    const response = await api.get(`/agent-extensions/${encodeURIComponent(type)}/config/`);
    return safeParse(agentExtensionConfigResponseSchema, response, `${type} Extension Settings`);
  }

  public async updateConfig(type: string, payload: UpdateAgentExtensionRequest) {
    const request = updateAgentExtensionRequestSchema.parse(payload);
    const response = await api.put(
      `/agent-extensions/${encodeURIComponent(type)}/config/`,
      request,
    );
    return safeParse(agentExtensionConfigResponseSchema, response, `${type} Extension Settings`);
  }

  public async test(type: string) {
    const response = await api.post(`/agent-extensions/${encodeURIComponent(type)}/test/`);
    return safeParse(agentExtensionTestResponseSchema, response, `${type} Extension Test`);
  }
}

import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  agentDefinitionSchema,
  agentTemplateListSchema,
  assistantMessageListSchema,
  assistantThreadListSchema,
  assistantThreadSchema,
  saveAgentDefinitionRequestSchema,
  sendMessageResultSchema,
  type AgentDefinition,
  type AssistantThread,
  type SaveAgentDefinitionRequest,
} from "@/types/assistant";
import { createLimitOffsetResponse } from "@trenova/shared/types/server";

const agentDefinitionListSchema = createLimitOffsetResponse(agentDefinitionSchema);

export class AssistantService {
  public async listThreads(limit = 50, offset = 0) {
    const response = await api.get(`/assistant/threads/?limit=${limit}&offset=${offset}`);
    return safeParse(assistantThreadListSchema, response, "Assistant Thread");
  }

  public async startThread(agentDefinitionId: string, title = "") {
    const response = await api.post("/assistant/threads/", { agentDefinitionId, title });
    return safeParse(assistantThreadSchema, response, "Assistant Thread");
  }

  public async getThread(id: AssistantThread["id"]) {
    const response = await api.get(`/assistant/threads/${id}/`);
    return safeParse(assistantThreadSchema, response, "Assistant Thread");
  }

  public async deleteThread(id: AssistantThread["id"]) {
    await api.delete(`/assistant/threads/${id}/`);
  }

  public async listMessages(threadId: AssistantThread["id"]) {
    const response = await api.get(`/assistant/threads/${threadId}/messages/`);
    return safeParse(assistantMessageListSchema, response, "Assistant Message");
  }

  /**
   * A refused turn comes back as a normal result with `refused` set, not as an
   * error: the turn was processed, recorded, and explained.
   */
  public async sendMessage(threadId: AssistantThread["id"], content: string) {
    const response = await api.post(`/assistant/threads/${threadId}/messages/`, { content });
    return safeParse(sendMessageResultSchema, response, "Assistant Reply");
  }
}

export class AgentDefinitionService {
  public async list(enabledOnly = false) {
    const suffix = enabledOnly ? "?enabledOnly=true" : "";
    const response = await api.get(`/agent-definitions/${suffix}`);
    return safeParse(agentDefinitionListSchema, response, "Agent");
  }

  public async get(id: AgentDefinition["id"]) {
    const response = await api.get(`/agent-definitions/${id}/`);
    return safeParse(agentDefinitionSchema, response, "Agent");
  }

  public async templates() {
    const response = await api.get("/agent-definitions/templates/");
    return safeParse(agentTemplateListSchema, response, "Agent Template");
  }

  public async create(payload: SaveAgentDefinitionRequest) {
    const request = saveAgentDefinitionRequestSchema.parse(payload);
    const response = await api.post("/agent-definitions/", request);
    return safeParse(agentDefinitionSchema, response, "Agent");
  }

  public async update(id: AgentDefinition["id"], payload: SaveAgentDefinitionRequest) {
    const request = saveAgentDefinitionRequestSchema.parse(payload);
    const response = await api.put(`/agent-definitions/${id}/`, request);
    return safeParse(agentDefinitionSchema, response, "Agent");
  }

  public async remove(id: AgentDefinition["id"]) {
    await api.delete(`/agent-definitions/${id}/`);
  }
}

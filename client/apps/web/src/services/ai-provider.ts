import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  aiProviderCatalogSchema,
  aiProviderSchema,
  saveAIProviderRequestSchema,
  testAIProviderResultSchema,
  type AIProvider,
  type SaveAIProviderRequest,
} from "@/types/ai-provider";

export class AIProviderService {
  public async catalog() {
    const response = await api.get("/ai-providers/catalog/");
    return safeParse(aiProviderCatalogSchema, response, "AI Provider Catalog");
  }

  public async create(payload: SaveAIProviderRequest) {
    const request = saveAIProviderRequestSchema.parse(payload);
    const response = await api.post("/ai-providers/", request);
    return safeParse(aiProviderSchema, response, "AI Provider");
  }

  public async update(id: AIProvider["id"], payload: SaveAIProviderRequest) {
    const request = saveAIProviderRequestSchema.parse(payload);
    const response = await api.put(`/ai-providers/${id}/`, request);
    return safeParse(aiProviderSchema, response, "AI Provider");
  }

  public async remove(id: AIProvider["id"]) {
    await api.delete(`/ai-providers/${id}/`);
  }

  /**
   * Issues a live call against the endpoint. The result reports whether the
   * schema was honoured, which is the failure mode that cannot be detected any
   * other way: several OpenAI-compatible servers accept a json_schema request
   * and silently ignore it.
   */
  public async test(id: AIProvider["id"]) {
    const response = await api.post(`/ai-providers/${id}/test/`);
    return safeParse(testAIProviderResultSchema, response, "AI Provider Test");
  }
}

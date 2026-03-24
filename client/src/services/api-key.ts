import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import type { ResourceCategory } from "@/lib/role-api";
import {
  apiKeyListSchema,
  apiKeySchema,
  apiKeySecretSchema,
  createApiKeyRequestSchema,
  updateApiKeyRequestSchema,
  type ApiKey,
  type CreateApiKeyRequest,
  type UpdateApiKeyRequest,
} from "@/types/api-key";

export class APIKeyService {
  public async list() {
    const response = await api.get("/api-keys/");
    return safeParse(apiKeyListSchema, response, "API Key");
  }

  public async get(id: ApiKey["id"]) {
    const response = await api.get(`/api-keys/${id}/`);
    return safeParse(apiKeySchema, response, "API Key");
  }

  public async create(payload: CreateApiKeyRequest) {
    const request = createApiKeyRequestSchema.parse(payload);
    const response = await api.post("/api-keys/", request);
    return safeParse(apiKeySecretSchema, response, "API Key Secret");
  }

  public async update(id: ApiKey["id"], payload: UpdateApiKeyRequest) {
    const request = updateApiKeyRequestSchema.parse(payload);
    const response = await api.put(`/api-keys/${id}/`, request);
    return safeParse(apiKeySchema, response, "API Key");
  }

  public async rotate(id: ApiKey["id"]) {
    const response = await api.post(`/api-keys/${id}/rotate/`);
    return safeParse(apiKeySecretSchema, response, "API Key Secret");
  }

  public async revoke(id: ApiKey["id"]) {
    const response = await api.post(`/api-keys/${id}/revoke/`);
    return safeParse(apiKeySchema, response, "API Key");
  }

  public async getAllowedResources(): Promise<ResourceCategory[]> {
    return api.get<ResourceCategory[]>("/api-keys/allowed-resources");
  }
}

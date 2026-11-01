import {
  ApiKeyAllowedResourcesDocument,
  ApiKeyDetailDocument,
  CreateApiKeyDocument,
  RevokeApiKeyDocument,
  RotateApiKeyDocument,
  UpdateApiKeyDocument,
  type ApiKeyInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { resourceCategoriesFromGraphQL, type ResourceCategory } from "@/lib/role-api";
import {
  apiKeySchema,
  apiKeySecretSchema,
  createApiKeyRequestSchema,
  updateApiKeyRequestSchema,
  type ApiKey,
  type ApiKeySecret,
  type CreateApiKeyRequest,
  type UpdateApiKeyRequest,
} from "@/types/api-key";

type RequestOptions = { signal?: AbortSignal };

function apiKeyInput(request: CreateApiKeyRequest): ApiKeyInput {
  return {
    name: request.name,
    description: request.description,
    expiresAt: request.expiresAt,
    permissions: request.permissions.map((permission) => ({
      resource: permission.resource,
      operations: permission.operations,
      dataScope: permission.dataScope,
    })),
  };
}

type IssuedApiKey = { apiKey: Record<string, unknown>; token: string };

function issuedApiKey(issued: IssuedApiKey): Promise<ApiKeySecret> {
  return safeParse(apiKeySecretSchema, { ...issued.apiKey, token: issued.token }, "API Key Secret");
}

export class APIKeyService {
  public async get(id: ApiKey["id"], options: RequestOptions = {}) {
    const data = await requestGraphQL<{ apiKeyDetail: unknown }>({
      document: ApiKeyDetailDocument,
      operationName: "ApiKeyDetail",
      variables: { id },
      signal: options.signal,
    });
    return safeParse(apiKeySchema, data.apiKeyDetail, "API Key");
  }

  public async create(payload: CreateApiKeyRequest) {
    const request = createApiKeyRequestSchema.parse(payload);
    const data = await requestGraphQL<{ createApiKey: IssuedApiKey }>({
      document: CreateApiKeyDocument,
      operationName: "CreateApiKey",
      variables: { input: apiKeyInput(request) },
    });
    return issuedApiKey(data.createApiKey);
  }

  public async update(id: ApiKey["id"], payload: UpdateApiKeyRequest) {
    const request = updateApiKeyRequestSchema.parse(payload);
    const data = await requestGraphQL<{ updateApiKey: unknown }>({
      document: UpdateApiKeyDocument,
      operationName: "UpdateApiKey",
      variables: { id, input: apiKeyInput(request) },
    });
    return safeParse(apiKeySchema, data.updateApiKey, "API Key");
  }

  public async rotate(id: ApiKey["id"]) {
    const data = await requestGraphQL<{ rotateApiKey: IssuedApiKey }>({
      document: RotateApiKeyDocument,
      operationName: "RotateApiKey",
      variables: { id },
    });
    return issuedApiKey(data.rotateApiKey);
  }

  public async revoke(id: ApiKey["id"]) {
    const data = await requestGraphQL<{ revokeApiKey: unknown }>({
      document: RevokeApiKeyDocument,
      operationName: "RevokeApiKey",
      variables: { id },
    });
    return safeParse(apiKeySchema, data.revokeApiKey, "API Key");
  }

  public async getAllowedResources(options: RequestOptions = {}): Promise<ResourceCategory[]> {
    const data = await requestGraphQL({
      document: ApiKeyAllowedResourcesDocument,
      operationName: "ApiKeyAllowedResources",
      signal: options.signal,
    });
    return resourceCategoriesFromGraphQL(data.apiKeyAllowedResources);
  }
}

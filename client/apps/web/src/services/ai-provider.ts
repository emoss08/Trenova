import {
  AiProviderCatalogDocument,
  CreateAiProviderDocument,
  DeleteAiProviderDocument,
  TestAiProviderDocument,
  UpdateAiProviderDocument,
  type AiProviderCatalogQuery,
  type SaveAiProviderInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  aiProviderCatalogSchema,
  saveAIProviderRequestSchema,
  testAIProviderResultSchema,
  type AIProvider,
  type SaveAIProviderRequest,
} from "@/types/ai-provider";

function decimalOrNull(value: number | null): string | null {
  return value === null ? null : String(value);
}

/** The editor's request as the save mutation takes it. An absent key keeps the stored one. */
export function toSaveAIProviderInput(payload: SaveAIProviderRequest): SaveAiProviderInput {
  const request = saveAIProviderRequestSchema.parse(payload);

  return {
    name: request.name,
    description: request.description,
    kind: request.kind,
    baseUrl: request.baseUrl,
    model: request.model,
    apiKey: request.apiKey ?? null,
    allowPrivateNetwork: request.allowPrivateNetwork,
    structuredOutputMode: request.structuredOutputMode,
    reasoningEffort: request.reasoningEffort,
    thinkingStyle: request.thinkingStyle,
    platform: request.platform,
    platformSettings: request.platformSettings,
    advisorEnabled: request.advisorEnabled,
    advisorModel: request.advisorModel,
    scripts: request.scripts,
    scriptWrites: request.scriptWrites,
    extraBody: request.extraBody,
    inputCostPerMillion: decimalOrNull(request.inputCostPerMillion),
    outputCostPerMillion: decimalOrNull(request.outputCostPerMillion),
    cacheReadCostPerMillion: decimalOrNull(request.cacheReadCostPerMillion),
    cacheWriteCostPerMillion: decimalOrNull(request.cacheWriteCostPerMillion),
    maxTokens: request.maxTokens,
    tasks: request.tasks,
    priority: request.priority,
    embeddingDimensions: request.embeddingDimensions,
    embeddingInputStyle: request.embeddingInputStyle,
    trusted: request.trusted,
    enabled: request.enabled,
    timeoutSeconds: request.timeoutSeconds,
    maxConcurrent: request.maxConcurrent,
    monthlyCapUsd: decimalOrNull(request.monthlyCapUsd),
    onCap: request.onCap,
    keepPreviousKey: request.keepPreviousKey,
    version: request.version,
  };
}

/** A preset's embedding style and platform are empty when it has none; the editor reads that as absent. */
function catalogForEditor(catalog: AiProviderCatalogQuery["aiProviderCatalog"]) {
  return {
    ...catalog,
    presets: catalog.presets.map((preset) => ({
      ...preset,
      embeddingInputStyle: preset.embeddingInputStyle || undefined,
      platform: preset.platform || undefined,
    })),
  };
}

export class AIProviderService {
  public async catalog(options?: { signal?: AbortSignal }) {
    const data = await requestGraphQL({
      document: AiProviderCatalogDocument,
      operationName: "AIProviderCatalog",
      signal: options?.signal,
    });
    return safeParse(
      aiProviderCatalogSchema,
      catalogForEditor(data.aiProviderCatalog),
      "AI Provider Catalog",
    );
  }

  public async create(payload: SaveAIProviderRequest) {
    const data = await requestGraphQL({
      document: CreateAiProviderDocument,
      operationName: "CreateAIProvider",
      variables: { input: toSaveAIProviderInput(payload) },
    });
    return data.createAIProvider;
  }

  public async update(id: AIProvider["id"], payload: SaveAIProviderRequest) {
    const data = await requestGraphQL({
      document: UpdateAiProviderDocument,
      operationName: "UpdateAIProvider",
      variables: { id, input: toSaveAIProviderInput(payload) },
    });
    return data.updateAIProvider;
  }

  public async remove(id: AIProvider["id"]) {
    await requestGraphQL({
      document: DeleteAiProviderDocument,
      operationName: "DeleteAIProvider",
      variables: { id },
    });
  }

  /**
   * Issues a live call against the endpoint. The result reports whether the
   * schema was honoured, which is the failure mode that cannot be detected any
   * other way: several OpenAI-compatible servers accept a json_schema request
   * and silently ignore it.
   */
  public async test(id: AIProvider["id"]) {
    const data = await requestGraphQL({
      document: TestAiProviderDocument,
      operationName: "TestAIProvider",
      variables: { id },
    });
    return safeParse(testAIProviderResultSchema, data.testAIProvider, "AI Provider Test");
  }
}

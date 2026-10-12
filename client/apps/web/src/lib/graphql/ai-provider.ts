import {
  AiProviderCardFieldsFragmentDoc,
  AiProviderCardsDocument,
  AiProviderDetailDocument,
  AiProviderLimitFieldsFragmentDoc,
  AiProviderLimitsDocument,
  AiProviderModelsDocument,
  AiProviderUsageDailyDocument,
  AiRoutePreviewDocument,
  PatchAiProviderDocument,
  ReorderAiProvidersDocument,
  TestAiProviderDraftDocument,
  type AiProviderCardFieldsFragment,
  type AiProviderLimitFieldsFragment,
  type AiProviderDraftTestInput,
  type AiProviderEndpointInput,
  type AiProviderModelsQuery,
  type AiProviderPatchInput,
  type AiProviderRoutingDraftInput,
  type AiProviderUsageDailyQuery,
  type AiRoutePreviewQuery,
  type TestAiProviderDraftMutation,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AIProviderRow = AiProviderCardFieldsFragment;
export type AIProviderTestOutcome = NonNullable<AIProviderRow["lastTest"]>;
/**
 * A provider's limits and key, asked for apart from its card so the many places that
 * list providers do not depend on them; only the providers tab and its editor read them.
 */
export type AIProviderLimits = Omit<AiProviderLimitFieldsFragment, " $fragmentName">;
export type AIProviderWithLimits = AIProviderRow & AIProviderLimits;
export type AIProviderKeyInfo = NonNullable<AIProviderLimits["apiKey"]>;
export type AIProviderModelOption = AiProviderModelsQuery["aiProviderModels"][number];
export type AIProviderDraftTestResult = TestAiProviderDraftMutation["testAIProviderDraft"];
export type AITaskRoute = AiRoutePreviewQuery["aiRoutePreview"][number];
export type AIProviderUsageDay = AiProviderUsageDailyQuery["aiUsageDaily"][number];
export type AIProviderEndpoint = AiProviderEndpointInput;
export type AIProviderDraftTest = AiProviderDraftTestInput;
export type AIProviderPatch = AiProviderPatchInput;
export type AIProviderRoutingDraft = AiProviderRoutingDraftInput;

type RequestOptions = { signal?: AbortSignal };

/**
 * Every provider of the organization, in routing order: the one a task
 * reaches first comes first. An organization holds a handful of endpoints,
 * never pages of them, so one request carries the whole set.
 */
export async function fetchAIProviders(options?: RequestOptions): Promise<AIProviderRow[]> {
  const data = await requestGraphQL({
    document: AiProviderCardsDocument,
    operationName: "AIProviderCards",
    variables: {
      input: {
        first: 100,
        sort: [
          { field: "priority", direction: "asc" },
          { field: "name", direction: "asc" },
        ],
      },
    },
    signal: options?.signal,
  });

  return data.aiProviders.edges.map((edge) =>
    getFragmentData(AiProviderCardFieldsFragmentDoc, edge.node),
  );
}

export async function fetchAIProvider(
  id: string,
  options?: RequestOptions,
): Promise<AIProviderWithLimits | null> {
  const data = await requestGraphQL({
    document: AiProviderDetailDocument,
    operationName: "AIProviderDetail",
    variables: { id },
    signal: options?.signal,
  });

  if (!data.aiProvider) {
    return null;
  }
  return {
    ...getFragmentData(AiProviderCardFieldsFragmentDoc, data.aiProvider),
    ...limitsOf(getFragmentData(AiProviderLimitFieldsFragmentDoc, data.aiProvider)),
  };
}

function limitsOf(fragment: AiProviderLimitFieldsFragment): AIProviderLimits {
  const { " $fragmentName": _marker, ...limits } = fragment;
  return limits;
}

/** Every provider's limits and key, by provider ID. */
export async function fetchAIProviderLimits(
  options?: RequestOptions,
): Promise<Map<string, AIProviderLimits>> {
  const data = await requestGraphQL({
    document: AiProviderLimitsDocument,
    operationName: "AIProviderLimits",
    variables: { input: { first: 100 } },
    signal: options?.signal,
  });

  return new Map(
    data.aiProviders.edges.map((edge) => {
      const limits = limitsOf(getFragmentData(AiProviderLimitFieldsFragmentDoc, edge.node));
      return [String(limits.id), limits] as const;
    }),
  );
}

/** The models an endpoint says it serves, asked of the endpoint itself. */
export async function fetchAIProviderModels(
  input: AIProviderEndpoint,
  options?: RequestOptions,
): Promise<AIProviderModelOption[]> {
  const data = await requestGraphQL({
    document: AiProviderModelsDocument,
    operationName: "AIProviderModels",
    variables: { input },
    signal: options?.signal,
  });

  return data.aiProviderModels;
}

/** Where each task goes now and where it would go with the draft saved. */
export async function fetchAIRoutePreview(
  draft: AIProviderRoutingDraft,
  options?: RequestOptions,
): Promise<AITaskRoute[]> {
  const data = await requestGraphQL({
    document: AiRoutePreviewDocument,
    operationName: "AIRoutePreview",
    variables: { draft },
    signal: options?.signal,
  });

  return data.aiRoutePreview;
}

/** One provider's calls and failures, day by day, oldest first. */
export async function fetchAIProviderUsageDaily(
  providerId: string,
  days: number,
  timezone: string,
  options?: RequestOptions,
): Promise<AIProviderUsageDay[]> {
  const data = await requestGraphQL({
    document: AiProviderUsageDailyDocument,
    operationName: "AIProviderUsageDaily",
    variables: { providerId, days, timezone },
    signal: options?.signal,
  });

  return data.aiUsageDaily;
}

/** Probes an unsaved endpoint once; nothing is recorded. */
export async function testAIProviderDraft(
  input: AIProviderDraftTest,
): Promise<AIProviderDraftTestResult> {
  const data = await requestGraphQL({
    document: TestAiProviderDraftDocument,
    operationName: "TestAIProviderDraft",
    variables: { input },
  });

  return data.testAIProviderDraft;
}

/** Sets the routing order: the first ID takes work first. */
export async function reorderAIProviders(ids: readonly string[]): Promise<AIProviderRow[]> {
  const data = await requestGraphQL({
    document: ReorderAiProvidersDocument,
    operationName: "ReorderAIProviders",
    variables: { ids: [...ids] },
  });

  return data.reorderAIProviders.map((provider) =>
    getFragmentData(AiProviderCardFieldsFragmentDoc, provider),
  );
}

/** Changes the fields given and leaves the rest as they are. */
export async function patchAIProvider(
  provider: Pick<AIProviderRow, "id" | "version">,
  input: AIProviderPatch,
): Promise<AIProviderRow> {
  const data = await requestGraphQL({
    document: PatchAiProviderDocument,
    operationName: "PatchAIProvider",
    variables: { id: provider.id, version: provider.version, input },
  });

  return getFragmentData(AiProviderCardFieldsFragmentDoc, data.patchAIProvider);
}

import {
  AiProviderCardFieldsFragmentDoc,
  AiProviderCardsDocument,
  AiProviderDetailDocument,
  type AiProviderCardFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AIProviderRow = AiProviderCardFieldsFragment;
export type AIProviderTestOutcome = NonNullable<AIProviderRow["lastTest"]>;

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
): Promise<AIProviderRow | null> {
  const data = await requestGraphQL({
    document: AiProviderDetailDocument,
    operationName: "AIProviderDetail",
    variables: { id },
    signal: options?.signal,
  });

  return data.aiProvider ? getFragmentData(AiProviderCardFieldsFragmentDoc, data.aiProvider) : null;
}

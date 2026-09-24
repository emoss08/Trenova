import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AiRetrievalFailedEntryTableDocument,
  AiRetrievalReindexEstimateDocument,
  AiRetrievalStatusDocument,
  AiRetrievalStatusFieldsFragmentDoc,
  ReindexAiRetrievalSourceDocument,
  UpdateAiRetrievalSettingsDocument,
  type AiRetrievalIndexStatus,
  type AiRetrievalPauseReason,
  type AiRetrievalReindexEstimateQuery,
  type AiRetrievalSettingsPatchInput,
  type AiRetrievalSourceType,
  type AiRetrievalStatusFieldsFragment,
  type AiRetrievalUnavailableReason,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export type AIRetrievalStatus = AiRetrievalStatusFieldsFragment;
export type AIRetrievalSource = AIRetrievalStatus["sources"][number];
export type AIRetrievalSettings = AIRetrievalStatus["settings"];
export type AIRetrievalAvailability = AIRetrievalStatus["availability"];
export type AIRetrievalModelChange = NonNullable<AIRetrievalStatus["modelChange"]>;
export type AIRetrievalReindexEstimate =
  AiRetrievalReindexEstimateQuery["aiRetrievalReindexEstimate"];
export type {
  AiRetrievalIndexStatus as AIRetrievalIndexStatus,
  AiRetrievalPauseReason as AIRetrievalPauseReason,
  AiRetrievalSettingsPatchInput as AIRetrievalSettingsPatch,
  AiRetrievalSourceType as AIRetrievalSourceType,
  AiRetrievalUnavailableReason as AIRetrievalUnavailableReason,
};

type RequestOptions = { signal?: AbortSignal };

export const AI_RETRIEVAL_FAILED_LIST_KEY = "ai-retrieval-failed-list";

/**
 * The sources whose indexing failed or is waiting to retry, newest attempt
 * first, for one source type or all of them. The server reads the thousand
 * most recent and pages, searches, filters and sorts them.
 */
export function createAIRetrievalFailedEntryTableGraphQLConfig(
  sourceType: AiRetrievalSourceType | null,
) {
  return defineDataTableGraphQLConfig({
    document: AiRetrievalFailedEntryTableDocument,
    operationName: "AIRetrievalFailedEntryTable",
    connectionKey: "aiRetrievalFailedEntryConnection",
    extraVariables: sourceType ? { sourceType } : undefined,
  });
}

export type AIRetrievalFailedEntryRow = DataTableConfigRow<
  ReturnType<typeof createAIRetrievalFailedEntryTableGraphQLConfig>
>;

export async function fetchAIRetrievalStatus(options?: RequestOptions): Promise<AIRetrievalStatus> {
  const data = await requestGraphQL({
    document: AiRetrievalStatusDocument,
    operationName: "AIRetrievalStatus",
    variables: {},
    signal: options?.signal,
  });

  return getFragmentData(AiRetrievalStatusFieldsFragmentDoc, data.aiRetrievalStatus);
}

export async function fetchAIRetrievalReindexEstimate(
  sourceType: AiRetrievalSourceType,
  options?: RequestOptions,
): Promise<AIRetrievalReindexEstimate> {
  const data = await requestGraphQL({
    document: AiRetrievalReindexEstimateDocument,
    operationName: "AIRetrievalReindexEstimate",
    variables: { sourceType },
    signal: options?.signal,
  });

  return data.aiRetrievalReindexEstimate;
}

export async function updateAIRetrievalSettings(
  input: AiRetrievalSettingsPatchInput,
): Promise<AIRetrievalStatus> {
  const data = await requestGraphQL({
    document: UpdateAiRetrievalSettingsDocument,
    operationName: "UpdateAIRetrievalSettings",
    variables: { input },
  });

  return getFragmentData(AiRetrievalStatusFieldsFragmentDoc, data.updateAIRetrievalSettings);
}

export async function reindexAIRetrievalSource(
  sourceType: AiRetrievalSourceType,
): Promise<AIRetrievalStatus> {
  const data = await requestGraphQL({
    document: ReindexAiRetrievalSourceDocument,
    operationName: "ReindexAIRetrievalSource",
    variables: { sourceType },
  });

  return getFragmentData(AiRetrievalStatusFieldsFragmentDoc, data.reindexAIRetrievalSource);
}

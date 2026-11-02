import {
  AiCorrectionDetailDocument,
  AiCorrectionDetailFieldsFragmentDoc,
  AiCorrectionFieldResultFieldsFragmentDoc,
  AiCorrectionTableDocument,
  AiCorrectionTableRowFieldsFragmentDoc,
  ExtractionAccuracyDocument,
  ExtractionFieldAccuracyFieldsFragmentDoc,
  ExtractionProviderTrendsDocument,
  ExtractionWeekAccuracyFieldsFragmentDoc,
  type AiCorrectionFieldResultFieldsFragment,
  type AiCorrectionTableRowFieldsFragment,
  type ExtractionAccuracyQuery,
  type ExtractionFieldAccuracyFieldsFragment,
  type ExtractionProviderTrendsQuery,
  type ExtractionWeekAccuracyFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

type RequestOptions = { signal?: AbortSignal };

export const EXTRACTION_ACCURACY_KEY = "extraction-accuracy";
export const EXTRACTION_PROVIDER_TRENDS_KEY = "extraction-provider-trends";
export const AI_CORRECTION_LIST_KEY = "ai-correction-list";
export const AI_CORRECTION_DETAIL_KEY = "ai-correction";

export type ExtractionWeekAccuracy = ExtractionWeekAccuracyFieldsFragment;
type RawProviderTrend =
  ExtractionProviderTrendsQuery["extractionProviderTrends"]["providers"][number];
export type ExtractionProviderTrend = Omit<RawProviderTrend, "weeks" | "checked" | "baseline"> & {
  weeks: ExtractionWeekAccuracy[];
  checked: ExtractionWeekAccuracy;
  baseline: ExtractionWeekAccuracy;
};
export type ExtractionProviderTrends = Omit<
  ExtractionProviderTrendsQuery["extractionProviderTrends"],
  "providers"
> & { providers: ExtractionProviderTrend[] };

export type ExtractionFieldAccuracy = ExtractionFieldAccuracyFieldsFragment;
export type AICorrectionFieldResult = AiCorrectionFieldResultFieldsFragment;

type AccuracyNode = ExtractionAccuracyQuery["extractionAccuracy"];

export type ExtractionAccuracy = Omit<AccuracyNode, "fields"> & {
  fields: ExtractionFieldAccuracy[];
};

export type AICorrectionDetail = AiCorrectionTableRowFieldsFragment & {
  fieldResults: AICorrectionFieldResult[];
};

export const aiCorrectionTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AiCorrectionTableDocument,
  operationName: "AICorrectionTable",
  connectionKey: "aiCorrections",
});

export type AICorrectionRow = DataTableConfigRow<typeof aiCorrectionTableGraphQLConfig>;

export async function fetchExtractionAccuracy(
  windowDays: number,
  options?: RequestOptions,
): Promise<ExtractionAccuracy> {
  const data = await requestGraphQL({
    document: ExtractionAccuracyDocument,
    operationName: "ExtractionAccuracy",
    variables: { windowDays },
    signal: options?.signal,
  });
  const accuracy = data.extractionAccuracy;

  return {
    ...accuracy,
    fields: accuracy.fields.map((field) =>
      getFragmentData(ExtractionFieldAccuracyFieldsFragmentDoc, field),
    ),
  };
}

export async function fetchAICorrection(
  id: string,
  options?: RequestOptions,
): Promise<AICorrectionDetail | null> {
  const data = await requestGraphQL({
    document: AiCorrectionDetailDocument,
    operationName: "AICorrectionDetail",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.aiCorrection) return null;

  const detail = getFragmentData(AiCorrectionDetailFieldsFragmentDoc, data.aiCorrection);
  return {
    ...getFragmentData(AiCorrectionTableRowFieldsFragmentDoc, detail),
    fieldResults: detail.fieldResults.map((result) =>
      getFragmentData(AiCorrectionFieldResultFieldsFragmentDoc, result),
    ),
  };
}

export async function fetchExtractionProviderTrends(
  options?: RequestOptions,
): Promise<ExtractionProviderTrends> {
  const data = await requestGraphQL({
    document: ExtractionProviderTrendsDocument,
    operationName: "ExtractionProviderTrends",
    variables: {},
    signal: options?.signal,
  });
  const trends = data.extractionProviderTrends;
  const week = (value: RawProviderTrend["checked"]) =>
    getFragmentData(ExtractionWeekAccuracyFieldsFragmentDoc, value);

  return {
    ...trends,
    providers: trends.providers.map((provider) => ({
      ...provider,
      weeks: provider.weeks.map(week),
      checked: week(provider.checked),
      baseline: week(provider.baseline),
    })),
  };
}

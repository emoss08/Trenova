import {
  AiCorrectionFieldResultFieldsFragmentDoc,
  ExtractionShadowReportDocument,
  ExtractionShadowResultDetailDocument,
  ExtractionShadowResultDetailFieldsFragmentDoc,
  ExtractionShadowResultTableDocument,
  ExtractionShadowResultTableRowFieldsFragmentDoc,
  ExtractionShadowSettingsDocument,
  ExtractionShadowSettingsFieldsFragmentDoc,
  ExtractionSnapshotFieldsFragmentDoc,
  UpdateExtractionShadowSettingsDocument,
  type ExtractionShadowReportQuery,
  type ExtractionShadowResultTableRowFieldsFragment,
  type ExtractionShadowSettingsFieldsFragment,
  type UpdateExtractionShadowSettingsInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";
import type { AICorrectionFieldResult, ExtractionSnapshot } from "./extraction-eval";

type RequestOptions = { signal?: AbortSignal };

export const EXTRACTION_SHADOW_SETTINGS_KEY = "extraction-shadow-settings";
export const EXTRACTION_SHADOW_REPORT_KEY = "extraction-shadow-report";
export const EXTRACTION_SHADOW_RESULT_LIST_KEY = "extraction-shadow-result-list";
export const EXTRACTION_SHADOW_RESULT_DETAIL_KEY = "extraction-shadow-result";

export type ExtractionShadowSettings = ExtractionShadowSettingsFieldsFragment;
export type ExtractionShadowReport = ExtractionShadowReportQuery["extractionShadowReport"];
export type ExtractionShadowResult = ExtractionShadowResultTableRowFieldsFragment;

export type ExtractionShadowResultDetail = ExtractionShadowResult & {
  predicted: ExtractionSnapshot | null;
  fieldResults: AICorrectionFieldResult[];
  baselineFieldResults: AICorrectionFieldResult[];
};

export const extractionShadowResultTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: ExtractionShadowResultTableDocument,
  operationName: "ExtractionShadowResultTable",
  connectionKey: "extractionShadowResults",
});

export type ExtractionShadowResultRow = DataTableConfigRow<
  typeof extractionShadowResultTableGraphQLConfig
>;

export async function fetchExtractionShadowSettings(
  options?: RequestOptions,
): Promise<ExtractionShadowSettings> {
  const data = await requestGraphQL({
    document: ExtractionShadowSettingsDocument,
    operationName: "ExtractionShadowSettings",
    variables: {},
    signal: options?.signal,
  });

  return getFragmentData(ExtractionShadowSettingsFieldsFragmentDoc, data.extractionShadowSettings);
}

export async function fetchExtractionShadowReport(
  windowDays: number,
  providerId: string | null,
  options?: RequestOptions,
): Promise<ExtractionShadowReport> {
  const data = await requestGraphQL({
    document: ExtractionShadowReportDocument,
    operationName: "ExtractionShadowReport",
    variables: { windowDays, providerId },
    signal: options?.signal,
  });

  return data.extractionShadowReport;
}

export async function fetchExtractionShadowResult(
  id: string,
  options?: RequestOptions,
): Promise<ExtractionShadowResultDetail | null> {
  const data = await requestGraphQL({
    document: ExtractionShadowResultDetailDocument,
    operationName: "ExtractionShadowResultDetail",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.extractionShadowResult) return null;

  const detail = getFragmentData(
    ExtractionShadowResultDetailFieldsFragmentDoc,
    data.extractionShadowResult,
  );
  return {
    ...getFragmentData(ExtractionShadowResultTableRowFieldsFragmentDoc, detail),
    predicted: detail.predicted
      ? getFragmentData(ExtractionSnapshotFieldsFragmentDoc, detail.predicted)
      : null,
    fieldResults: detail.fieldResults.map((result) =>
      getFragmentData(AiCorrectionFieldResultFieldsFragmentDoc, result),
    ),
    baselineFieldResults: detail.baselineFieldResults.map((result) =>
      getFragmentData(AiCorrectionFieldResultFieldsFragmentDoc, result),
    ),
  };
}

export async function updateExtractionShadowSettings(
  input: UpdateExtractionShadowSettingsInput,
): Promise<ExtractionShadowSettings> {
  const data = await requestGraphQL({
    document: UpdateExtractionShadowSettingsDocument,
    operationName: "UpdateExtractionShadowSettings",
    variables: { input },
  });

  return getFragmentData(
    ExtractionShadowSettingsFieldsFragmentDoc,
    data.updateExtractionShadowSettings,
  );
}

import {
  AiCorrectionDetailDocument,
  AiCorrectionDetailFieldsFragmentDoc,
  AiCorrectionFieldResultFieldsFragmentDoc,
  AiCorrectionTableDocument,
  AiCorrectionTableRowFieldsFragmentDoc,
  CancelExtractionEvalRunDocument,
  DataTablePageInfoFieldsFragmentDoc,
  DeleteExtractionEvalCaseDocument,
  ExtractionAccuracyDocument,
  ExtractionEvalCaseDetailDocument,
  ExtractionEvalCaseTableDocument,
  ExtractionEvalCaseTableRowFieldsFragmentDoc,
  ExtractionEvalResultDetailDocument,
  ExtractionEvalResultTableDocument,
  ExtractionEvalResultTableRowFieldsFragmentDoc,
  ExtractionEvalRunDetailDocument,
  ExtractionEvalRunDetailFieldsFragmentDoc,
  ExtractionEvalRunFieldsFragmentDoc,
  ExtractionEvalRunTableDocument,
  ExtractionFieldAccuracyFieldsFragmentDoc,
  ExtractionSnapshotFieldsFragmentDoc,
  PromoteAiCorrectionDocument,
  StartExtractionEvalRunDocument,
  UpdateExtractionEvalCaseDocument,
  type AiCorrectionFieldResultFieldsFragment,
  type AiCorrectionTableRowFieldsFragment,
  type ExtractionAccuracyQuery,
  type ExtractionEvalCaseTableRowFieldsFragment,
  type ExtractionEvalResultTableRowFieldsFragment,
  type ExtractionEvalRunFieldsFragment,
  type ExtractionFieldAccuracyFieldsFragment,
  type ExtractionSnapshotFieldsFragment,
  type PromoteAiCorrectionInput,
  type StartExtractionEvalRunInput,
  type UpdateExtractionEvalCaseInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

type RequestOptions = { signal?: AbortSignal };

export const EXTRACTION_ACCURACY_KEY = "extraction-accuracy";
export const AI_CORRECTION_LIST_KEY = "ai-correction-list";
export const AI_CORRECTION_DETAIL_KEY = "ai-correction";
export const EXTRACTION_EVAL_CASE_LIST_KEY = "extraction-eval-case-list";
export const EXTRACTION_EVAL_CASE_DETAIL_KEY = "extraction-eval-case";
export const EXTRACTION_EVAL_RUN_LIST_KEY = "extraction-eval-run-list";
export const EXTRACTION_EVAL_RUN_DETAIL_KEY = "extraction-eval-run";
export const EXTRACTION_EVAL_RESULT_LIST_KEY = "extraction-eval-result-list";
export const EXTRACTION_EVAL_RESULT_DETAIL_KEY = "extraction-eval-result";

export type ExtractionFieldAccuracy = ExtractionFieldAccuracyFieldsFragment;
export type ExtractionSnapshot = ExtractionSnapshotFieldsFragment;
export type AICorrectionFieldResult = AiCorrectionFieldResultFieldsFragment;
export type ExtractionEvalRun = ExtractionEvalRunFieldsFragment;

type AccuracyNode = ExtractionAccuracyQuery["extractionAccuracy"];

export type ExtractionAccuracy = Omit<AccuracyNode, "fields" | "recentRuns"> & {
  fields: ExtractionFieldAccuracy[];
  recentRuns: ExtractionEvalRun[];
};

export type ExtractionEvalRunDetail = ExtractionEvalRun & {
  fieldAccuracy: ExtractionFieldAccuracy[];
};

export type AICorrectionDetail = AiCorrectionTableRowFieldsFragment & {
  fieldResults: AICorrectionFieldResult[];
};

export type ExtractionEvalCaseDetail = ExtractionEvalCaseTableRowFieldsFragment & {
  expected: ExtractionSnapshot;
};

export type ExtractionEvalResultDetail = ExtractionEvalResultTableRowFieldsFragment & {
  fieldResults: AICorrectionFieldResult[];
};

export const aiCorrectionTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AiCorrectionTableDocument,
  operationName: "AICorrectionTable",
  connectionKey: "aiCorrections",
});

export type AICorrectionRow = DataTableConfigRow<typeof aiCorrectionTableGraphQLConfig>;

export const extractionEvalCaseTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: ExtractionEvalCaseTableDocument,
  operationName: "ExtractionEvalCaseTable",
  connectionKey: "extractionEvalCases",
});

export type ExtractionEvalCaseRow = DataTableConfigRow<typeof extractionEvalCaseTableGraphQLConfig>;

export const extractionEvalRunTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: ExtractionEvalRunTableDocument,
  operationName: "ExtractionEvalRunTable",
  connectionKey: "extractionEvalRuns",
});

export type ExtractionEvalRunRow = DataTableConfigRow<typeof extractionEvalRunTableGraphQLConfig>;

export function createExtractionEvalResultTableGraphQLConfig(runId: string) {
  return defineDataTableGraphQLConfig({
    document: ExtractionEvalResultTableDocument,
    operationName: "ExtractionEvalResultTable",
    connectionKey: "extractionEvalResults",
    extraVariables: { runId },
  });
}

export type ExtractionEvalResultRow = DataTableConfigRow<
  ReturnType<typeof createExtractionEvalResultTableGraphQLConfig>
>;

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
    recentRuns: accuracy.recentRuns.map((run) =>
      getFragmentData(ExtractionEvalRunFieldsFragmentDoc, run),
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

export async function fetchExtractionEvalCase(
  id: string,
  options?: RequestOptions,
): Promise<ExtractionEvalCaseDetail | null> {
  const data = await requestGraphQL({
    document: ExtractionEvalCaseDetailDocument,
    operationName: "ExtractionEvalCaseDetail",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.extractionEvalCase) return null;

  return {
    ...getFragmentData(ExtractionEvalCaseTableRowFieldsFragmentDoc, data.extractionEvalCase),
    expected: getFragmentData(
      ExtractionSnapshotFieldsFragmentDoc,
      data.extractionEvalCase.expected,
    ),
  };
}

export async function fetchExtractionEvalRun(
  id: string,
  options?: RequestOptions,
): Promise<ExtractionEvalRunDetail | null> {
  const data = await requestGraphQL({
    document: ExtractionEvalRunDetailDocument,
    operationName: "ExtractionEvalRunDetail",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.extractionEvalRun) return null;

  const detail = getFragmentData(ExtractionEvalRunDetailFieldsFragmentDoc, data.extractionEvalRun);
  return {
    ...getFragmentData(ExtractionEvalRunFieldsFragmentDoc, detail),
    fieldAccuracy: detail.fieldAccuracy.map((field) =>
      getFragmentData(ExtractionFieldAccuracyFieldsFragmentDoc, field),
    ),
  };
}

export async function fetchExtractionEvalResult(
  id: string,
  options?: RequestOptions,
): Promise<ExtractionEvalResultDetail | null> {
  const data = await requestGraphQL({
    document: ExtractionEvalResultDetailDocument,
    operationName: "ExtractionEvalResultDetail",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.extractionEvalResult) return null;

  return {
    ...getFragmentData(ExtractionEvalResultTableRowFieldsFragmentDoc, data.extractionEvalResult),
    fieldResults: data.extractionEvalResult.fieldResults.map((result) =>
      getFragmentData(AiCorrectionFieldResultFieldsFragmentDoc, result),
    ),
  };
}

export async function promoteAICorrection(input: PromoteAiCorrectionInput) {
  const data = await requestGraphQL({
    document: PromoteAiCorrectionDocument,
    operationName: "PromoteAICorrection",
    variables: { input },
  });

  return getFragmentData(ExtractionEvalCaseTableRowFieldsFragmentDoc, data.promoteAICorrection);
}

export async function updateExtractionEvalCase(id: string, input: UpdateExtractionEvalCaseInput) {
  const data = await requestGraphQL({
    document: UpdateExtractionEvalCaseDocument,
    operationName: "UpdateExtractionEvalCase",
    variables: { id, input },
  });

  return getFragmentData(
    ExtractionEvalCaseTableRowFieldsFragmentDoc,
    data.updateExtractionEvalCase,
  );
}

export async function deleteExtractionEvalCase(id: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: DeleteExtractionEvalCaseDocument,
    operationName: "DeleteExtractionEvalCase",
    variables: { id },
  });

  return data.deleteExtractionEvalCase;
}

export async function startExtractionEvalRun(input: StartExtractionEvalRunInput) {
  const data = await requestGraphQL({
    document: StartExtractionEvalRunDocument,
    operationName: "StartExtractionEvalRun",
    variables: { input },
  });

  return getFragmentData(ExtractionEvalRunFieldsFragmentDoc, data.startExtractionEvalRun);
}

export async function cancelExtractionEvalRun(id: string) {
  const data = await requestGraphQL({
    document: CancelExtractionEvalRunDocument,
    operationName: "CancelExtractionEvalRun",
    variables: { id },
  });

  return getFragmentData(ExtractionEvalRunFieldsFragmentDoc, data.cancelExtractionEvalRun);
}

export type ExtractionEvalResult = ExtractionEvalResultTableRowFieldsFragment;

const RESULT_PAGE_SIZE = 100;
const MAX_RESULT_PAGES = 5;

/** A run's results in the order they ran; a run holds at most 500 cases, read a page at a time. */
export async function fetchExtractionEvalResults(
  runId: string,
  options?: RequestOptions,
): Promise<ExtractionEvalResult[]> {
  const results: ExtractionEvalResult[] = [];
  let after: string | null | undefined;

  for (let page = 0; page < MAX_RESULT_PAGES; page++) {
    const data = await requestGraphQL({
      document: ExtractionEvalResultTableDocument,
      operationName: "ExtractionEvalResultTable",
      variables: {
        runId,
        includeTotalCount: false,
        input: {
          first: RESULT_PAGE_SIZE,
          after,
          sort: [{ field: "ordinal", direction: "asc" }],
        },
      },
      signal: options?.signal,
    });
    const connection = data.extractionEvalResults;
    for (const edge of connection.edges) {
      results.push(getFragmentData(ExtractionEvalResultTableRowFieldsFragmentDoc, edge.node));
    }
    const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);
    if (!pageInfo.hasNextPage || !pageInfo.endCursor) {
      break;
    }
    after = pageInfo.endCursor;
  }

  return results;
}

import {
  CancelReportRunDocument,
  CreateReportDefinitionDocument,
  CreateReportScheduleDocument,
  DeleteReportDefinitionDocument,
  DeleteReportScheduleDocument,
  ForkCannedReportDocument,
  ReportDefinitionFieldsFragmentDoc,
  ReportDefinitionsTableDocument,
  ReportRunFieldsFragmentDoc,
  ReportRunsTableDocument,
  ReportScheduleFieldsFragmentDoc,
  ResetCannedForkDocument,
  RunReportDocument,
  UpdateReportDefinitionDocument,
  UpdateReportScheduleDocument,
  type CannedReportsQuery,
  type CreateReportScheduleInput,
  type ForkCannedReportInput,
  type PreviewReportQuery,
  type ReportCatalogQuery,
  type ReportDefinitionFieldsFragment,
  type ReportDefinitionRevisionsQuery,
  type ReportDefinitionsTableQueryVariables,
  type ReportIrInput,
  type ReportRunFieldsFragment,
  type ReportRunsFilterInput,
  type ReportRunsTableQueryVariables,
  type ReportScheduleFieldsFragment,
  type RunReportInput,
  type SaveReportDefinitionInput,
  type UpdateReportDefinitionInput,
  type UpdateReportScheduleInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/generated";
import { API_BASE_URL } from "@/lib/constants";
import { requestGraphQL } from "@/lib/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";

export type ReportCatalog = ReportCatalogQuery["reportCatalog"];
export type ReportCatalogEntity = ReportCatalog["entities"][number];
export type ReportCatalogField = ReportCatalogEntity["fields"][number];
export type ReportCatalogEdge = ReportCatalogEntity["edges"][number];
export type CannedReport = CannedReportsQuery["cannedReports"][number];
export type ReportDefinition = ReportDefinitionFieldsFragment;
export type ReportDefinitionRevision =
  ReportDefinitionRevisionsQuery["reportDefinitionRevisions"][number];
export type ReportRun = ReportRunFieldsFragment;
export type ReportSchedule = ReportScheduleFieldsFragment;
export type ReportPreview = PreviewReportQuery["previewReport"];
export type ReportPreviewColumn = ReportPreview["columns"][number];

export const reportDefinitionsTableGraphQLConfig = defineDataTableGraphQLConfig<
  ReportDefinition & Record<string, unknown>,
  ReportDefinitionsTableQueryVariables
>({
  document: ReportDefinitionsTableDocument,
  operationName: "ReportDefinitionsTable",
  connectionKey: "reportDefinitions",
});

export function reportRunsTableGraphQLConfig(filter?: ReportRunsFilterInput) {
  return defineDataTableGraphQLConfig<
    ReportRun & Record<string, unknown>,
    ReportRunsTableQueryVariables
  >({
    document: ReportRunsTableDocument,
    operationName: "ReportRunsTable",
    connectionKey: "reportRuns",
    extraVariables: filter ? { filter } : undefined,
  });
}

export function reportRunDownloadUrl(runId: string): string {
  return `${API_BASE_URL}/reports/runs/${encodeURIComponent(runId)}/download/`;
}

export async function createReportDefinition(
  input: SaveReportDefinitionInput,
): Promise<ReportDefinition> {
  const data = await requestGraphQL({
    document: CreateReportDefinitionDocument,
    operationName: "CreateReportDefinition",
    variables: { input },
  });

  return getFragmentData(ReportDefinitionFieldsFragmentDoc, data.createReportDefinition);
}

export async function updateReportDefinition(
  input: UpdateReportDefinitionInput,
): Promise<ReportDefinition> {
  const data = await requestGraphQL({
    document: UpdateReportDefinitionDocument,
    operationName: "UpdateReportDefinition",
    variables: { input },
  });

  return getFragmentData(ReportDefinitionFieldsFragmentDoc, data.updateReportDefinition);
}

export async function deleteReportDefinition(id: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: DeleteReportDefinitionDocument,
    operationName: "DeleteReportDefinition",
    variables: { id },
  });

  return data.deleteReportDefinition;
}

export async function forkCannedReport(input: ForkCannedReportInput): Promise<ReportDefinition> {
  const data = await requestGraphQL({
    document: ForkCannedReportDocument,
    operationName: "ForkCannedReport",
    variables: { input },
  });

  return getFragmentData(ReportDefinitionFieldsFragmentDoc, data.forkCannedReport);
}

export async function resetCannedFork(id: string): Promise<ReportDefinition> {
  const data = await requestGraphQL({
    document: ResetCannedForkDocument,
    operationName: "ResetCannedFork",
    variables: { id },
  });

  return getFragmentData(ReportDefinitionFieldsFragmentDoc, data.resetCannedFork);
}

export async function runReport(input: RunReportInput): Promise<ReportRun> {
  const data = await requestGraphQL({
    document: RunReportDocument,
    operationName: "RunReport",
    variables: { input },
  });

  return getFragmentData(ReportRunFieldsFragmentDoc, data.runReport);
}

export async function cancelReportRun(id: string): Promise<ReportRun> {
  const data = await requestGraphQL({
    document: CancelReportRunDocument,
    operationName: "CancelReportRun",
    variables: { id },
  });

  return getFragmentData(ReportRunFieldsFragmentDoc, data.cancelReportRun);
}

export async function createReportSchedule(
  input: CreateReportScheduleInput,
): Promise<ReportSchedule> {
  const data = await requestGraphQL({
    document: CreateReportScheduleDocument,
    operationName: "CreateReportSchedule",
    variables: { input },
  });

  return getFragmentData(ReportScheduleFieldsFragmentDoc, data.createReportSchedule);
}

export async function updateReportSchedule(
  input: UpdateReportScheduleInput,
): Promise<ReportSchedule> {
  const data = await requestGraphQL({
    document: UpdateReportScheduleDocument,
    operationName: "UpdateReportSchedule",
    variables: { input },
  });

  return getFragmentData(ReportScheduleFieldsFragmentDoc, data.updateReportSchedule);
}

export async function deleteReportSchedule(id: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: DeleteReportScheduleDocument,
    operationName: "DeleteReportSchedule",
    variables: { id },
  });

  return data.deleteReportSchedule;
}

export type { ReportIrInput, ReportRunsFilterInput };

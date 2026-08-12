import {
  CancelReportRunDocument,
  CreateReportDashboardDocument,
  CreateReportDefinitionDocument,
  CreateReportScheduleDocument,
  CreateReportViewDocument,
  DeleteReportDashboardDocument,
  DeleteReportDefinitionDocument,
  DeleteReportScheduleDocument,
  DeleteReportViewDocument,
  ForkCannedReportDocument,
  ReportDashboardFieldsFragmentDoc,
  ReportDefinitionFieldsFragmentDoc,
  ReportDefinitionsTableDocument,
  ReportRunFieldsFragmentDoc,
  ReportRunsTableDocument,
  ReportScheduleFieldsFragmentDoc,
  ReportViewFieldsFragmentDoc,
  ResetCannedForkDocument,
  RunReportDocument,
  UpdateReportDashboardDocument,
  UpdateReportDefinitionDocument,
  UpdateReportScheduleDocument,
  UpdateReportViewDocument,
  type CannedReportsQuery,
  type CreateReportScheduleInput,
  type CreateReportViewInput,
  type ForkCannedReportInput,
  type ReportCatalogQuery,
  type ReportDashboardFieldsFragment,
  type ReportPreviewFieldsFragment,
  type SaveReportDashboardInput,
  type UpdateReportDashboardInput,
  type ReportDefinitionFieldsFragment,
  type ReportDefinitionRevisionsQuery,
  type ReportDefinitionsTableQueryVariables,
  type ReportIrInput,
  type ReportRunFieldsFragment,
  type ReportRunsFilterInput,
  type ReportRunsTableQueryVariables,
  type ReportScheduleFieldsFragment,
  type ReportViewFieldsFragment,
  type RunReportInput,
  type SaveReportDefinitionInput,
  type UpdateReportDefinitionInput,
  type UpdateReportScheduleInput,
  type UpdateReportViewInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { withCsrfHeader } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

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
export type ReportView = ReportViewFieldsFragment;
export type ReportPreview = ReportPreviewFieldsFragment;
export type ReportPreviewColumn = ReportPreview["columns"][number];
export type ReportDashboard = ReportDashboardFieldsFragment;

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

export type DashboardExportRequest = {
  filterValues?: Record<string, unknown>;
  paramValues?: Record<string, unknown>;
  /** Per-tile ad-hoc filters, keyed by tile id. */
  tileFilters?: Record<string, { filters: unknown[]; values: Record<string, unknown> }>;
};

/**
 * Exports every tile of a dashboard as one workbook. The server re-runs each
 * tile with the caller's own permissions, so the request carries the viewer's
 * filter choices rather than a pre-resolved query.
 */
export async function exportDashboard(
  dashboardId: string,
  body: DashboardExportRequest,
): Promise<{ blob: Blob; fileName: string }> {
  // A workbook is binary, so this bypasses the JSON api helper while keeping
  // its credential and CSRF handling.
  const endpoint = `/reports/dashboards/${encodeURIComponent(dashboardId)}/export/`;
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    method: "POST",
    credentials: "include",
    body: JSON.stringify(body),
    headers: await withCsrfHeader(
      "POST",
      { "Content-Type": "application/json" },
      endpoint,
    ),
  });

  if (!response.ok) {
    throw new Error("The dashboard could not be exported");
  }

  return {
    blob: await response.blob(),
    fileName: filenameFromDisposition(response.headers.get("Content-Disposition")),
  };
}

function filenameFromDisposition(header: string | null): string {
  const match = header?.match(/filename="?([^"]+)"?/i);
  return match?.[1] ?? "dashboard.xlsx";
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

export async function createReportDashboard(
  input: SaveReportDashboardInput,
): Promise<ReportDashboard> {
  const data = await requestGraphQL({
    document: CreateReportDashboardDocument,
    operationName: "CreateReportDashboard",
    variables: { input },
  });

  return getFragmentData(ReportDashboardFieldsFragmentDoc, data.createReportDashboard);
}

export async function updateReportDashboard(
  input: UpdateReportDashboardInput,
): Promise<ReportDashboard> {
  const data = await requestGraphQL({
    document: UpdateReportDashboardDocument,
    operationName: "UpdateReportDashboard",
    variables: { input },
  });

  return getFragmentData(ReportDashboardFieldsFragmentDoc, data.updateReportDashboard);
}

export async function deleteReportDashboard(id: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: DeleteReportDashboardDocument,
    operationName: "DeleteReportDashboard",
    variables: { id },
  });

  return data.deleteReportDashboard;
}

export async function createReportView(input: CreateReportViewInput): Promise<ReportView> {
  const data = await requestGraphQL({
    document: CreateReportViewDocument,
    operationName: "CreateReportView",
    variables: { input },
  });

  return getFragmentData(ReportViewFieldsFragmentDoc, data.createReportView);
}

export async function updateReportView(input: UpdateReportViewInput): Promise<ReportView> {
  const data = await requestGraphQL({
    document: UpdateReportViewDocument,
    operationName: "UpdateReportView",
    variables: { input },
  });

  return getFragmentData(ReportViewFieldsFragmentDoc, data.updateReportView);
}

export async function deleteReportView(id: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: DeleteReportViewDocument,
    operationName: "DeleteReportView",
    variables: { id },
  });

  return data.deleteReportView;
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

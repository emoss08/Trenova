import {
  CannedReportsDocument,
  DrillThroughReportDocument,
  PreviewReportDocument,
  ReportCatalogDocument,
  ReportDashboardByIdDocument,
  ReportDashboardsDocument,
  ReportDefinitionByIdDocument,
  ReportDefinitionRevisionsDocument,
  ReportDefinitionsTableDocument,
  ReportRunByIdDocument,
  ReportSchedulesDocument,
  ReportViewsDocument,
  type ReportDrillInput,
  type ReportIrInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const reports = createQueryKeys("reports", {
  catalog: () => ({
    queryKey: ["catalog"],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: ReportCatalogDocument,
        operationName: "ReportCatalog",
        signal,
      }),
  }),
  canned: () => ({
    queryKey: ["canned"],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: CannedReportsDocument,
        operationName: "CannedReports",
        signal,
      }),
  }),
  definitionList: (search: string) => ({
    queryKey: [search],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: ReportDefinitionsTableDocument,
        operationName: "ReportDefinitionsTable",
        variables: { input: { first: 100, query: search || undefined } },
        signal,
      }),
  }),
  definition: (id: string) => ({
    queryKey: [id],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: ReportDefinitionByIdDocument,
        operationName: "ReportDefinitionById",
        variables: { id },
        signal,
      }),
  }),
  revisions: (definitionId: string, limit?: number) => ({
    queryKey: [definitionId, limit ?? 0],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: ReportDefinitionRevisionsDocument,
        operationName: "ReportDefinitionRevisions",
        variables: { definitionId, limit },
        signal,
      }),
  }),
  run: (id: string) => ({
    queryKey: [id],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: ReportRunByIdDocument,
        operationName: "ReportRunById",
        variables: { id },
        signal,
      }),
  }),
  schedules: (definitionId?: string) => ({
    queryKey: [definitionId ?? "all"],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: ReportSchedulesDocument,
        operationName: "ReportSchedules",
        variables: { definitionId },
        signal,
      }),
  }),
  views: (definitionId?: string) => ({
    queryKey: [definitionId ?? "all"],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: ReportViewsDocument,
        operationName: "ReportViews",
        variables: { definitionId },
        signal,
      }),
  }),
  preview: (definition: ReportIrInput, params?: Record<string, unknown>, supersede = false) => ({
    queryKey: [definition, params, supersede],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: PreviewReportDocument,
        operationName: "PreviewReport",
        variables: { definition, params, supersede },
        signal,
      }),
  }),
  drillThrough: (input: ReportDrillInput) => ({
    queryKey: [input],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: DrillThroughReportDocument,
        operationName: "DrillThroughReport",
        variables: { input },
        signal,
      }),
  }),
  dashboards: () => ({
    queryKey: ["dashboards"],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: ReportDashboardsDocument,
        operationName: "ReportDashboards",
        signal,
      }),
  }),
  dashboard: (id: string) => ({
    queryKey: [id],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: ReportDashboardByIdDocument,
        operationName: "ReportDashboardById",
        variables: { id },
        signal,
      }),
  }),
});

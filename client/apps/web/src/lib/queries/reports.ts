import {
  CannedReportsDocument,
  PreviewReportDocument,
  ReportCatalogDocument,
  ReportDefinitionByIdDocument,
  ReportDefinitionRevisionsDocument,
  ReportDefinitionsTableDocument,
  ReportRunByIdDocument,
  ReportSchedulesDocument,
  type ReportIrInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const reports = createQueryKeys("reports", {
  catalog: () => ({
    queryKey: ["catalog"],
    queryFn: async () =>
      requestGraphQL({
        document: ReportCatalogDocument,
        operationName: "ReportCatalog",
      }),
  }),
  canned: () => ({
    queryKey: ["canned"],
    queryFn: async () =>
      requestGraphQL({
        document: CannedReportsDocument,
        operationName: "CannedReports",
      }),
  }),
  definitionList: (search: string) => ({
    queryKey: [search],
    queryFn: async () =>
      requestGraphQL({
        document: ReportDefinitionsTableDocument,
        operationName: "ReportDefinitionsTable",
        variables: { input: { first: 100, query: search || undefined } },
      }),
  }),
  definition: (id: string) => ({
    queryKey: [id],
    queryFn: async () =>
      requestGraphQL({
        document: ReportDefinitionByIdDocument,
        operationName: "ReportDefinitionById",
        variables: { id },
      }),
  }),
  revisions: (definitionId: string, limit?: number) => ({
    queryKey: [definitionId, limit ?? 0],
    queryFn: async () =>
      requestGraphQL({
        document: ReportDefinitionRevisionsDocument,
        operationName: "ReportDefinitionRevisions",
        variables: { definitionId, limit },
      }),
  }),
  run: (id: string) => ({
    queryKey: [id],
    queryFn: async () =>
      requestGraphQL({
        document: ReportRunByIdDocument,
        operationName: "ReportRunById",
        variables: { id },
      }),
  }),
  schedules: (definitionId?: string) => ({
    queryKey: [definitionId ?? "all"],
    queryFn: async () =>
      requestGraphQL({
        document: ReportSchedulesDocument,
        operationName: "ReportSchedules",
        variables: { definitionId },
      }),
  }),
  preview: (definition: ReportIrInput, params?: Record<string, unknown>) => ({
    queryKey: [definition, params],
    queryFn: async () =>
      requestGraphQL({
        document: PreviewReportDocument,
        operationName: "PreviewReport",
        variables: { definition, params },
      }),
  }),
});

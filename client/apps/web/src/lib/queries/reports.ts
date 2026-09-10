import {
  CannedReportsDocument,
  DrillThroughReportDocument,
  PreviewReportDocument,
  ReportCatalogDocument,
  ReportDashboardByIdDocument,
  ReportDashboardsDocument,
  ReportDefinitionByIdDocument,
  ReportDefinitionRevisionsDocument,
  DataTablePageInfoFieldsFragmentDoc,
  ReportDefinitionsTableDocument,
  ReportRunByIdDocument,
  ReportSchedulesDocument,
  ReportViewsDocument,
  type ReportDefinitionsTableQuery,
  type ReportDrillInput,
  type ReportIrInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/** Reports per page in the library grid. Two full rows on a wide screen. */
const REPORT_DEFINITION_PAGE_SIZE = 24;

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
  // Key space only. The library is cursor-paged, so its options are built by
  // reportDefinitionsInfiniteQuery below — the query-key factory cannot carry
  // the paging fields.
  definitionList: (search: string) => ({
    queryKey: [search],
  }),
  // Key space only, as above: the picker pages this connection with
  // useInfiniteQuery and owns its own queryFn.
  definitionOptions: (search: string) => ({
    queryKey: [search],
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

/**
 * The report library, one page at a time. The grid and the route's prefetch
 * both read it, so the cursor plumbing lives here rather than in either.
 */
export function reportDefinitionsInfiniteQuery(search: string) {
  return {
    queryKey: reports.definitionList(search).queryKey,
    queryFn: async ({ signal, pageParam }: { signal: AbortSignal; pageParam?: unknown }) =>
      requestGraphQL({
        document: ReportDefinitionsTableDocument,
        operationName: "ReportDefinitionsTable",
        variables: {
          input: {
            first: REPORT_DEFINITION_PAGE_SIZE,
            after: (pageParam as string | null) ?? null,
            query: search || undefined,
            sort: [{ field: "name", direction: "asc" }],
          },
          includeTotalCount: false,
        },
        signal,
      }),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage: ReportDefinitionsTableQuery) => {
      const { hasNextPage, endCursor } = getFragmentData(
        DataTablePageInfoFieldsFragmentDoc,
        lastPage.reportDefinitions.pageInfo,
      );
      return hasNextPage && endCursor ? endCursor : undefined;
    },
  };
}

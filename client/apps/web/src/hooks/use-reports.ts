import { getFragmentData } from "@trenova/graphql/generated";
import {
  ReportDashboardFieldsFragmentDoc,
  ReportDefinitionFieldsFragmentDoc,
  ReportPreviewFieldsFragmentDoc,
  ReportRunFieldsFragmentDoc,
  ReportScheduleFieldsFragmentDoc,
  ReportViewFieldsFragmentDoc,
  type ReportDrillInput,
  type ReportIrInput,
} from "@trenova/graphql/generated/graphql";
import {
  cancelReportRun,
  createReportDashboard,
  createReportDefinition,
  createReportSchedule,
  createReportView,
  deleteReportDashboard,
  deleteReportDefinition,
  deleteReportSchedule,
  deleteReportView,
  forkCannedReport,
  reportRunDownloadUrl,
  resetCannedFork,
  runReport,
  updateReportDashboard,
  updateReportDefinition,
  updateReportSchedule,
  updateReportView,
  type ReportRun,
} from "@/lib/graphql/reports";
import { queries } from "@/lib/queries";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const CATALOG_STALE_TIME = 5 * 60_000;
const RUN_POLL_INTERVAL = 3_000;

export const REPORT_RUN_LIST_QUERY_KEY = "report-run-list";

const ACTIVE_RUN_STATUSES = new Set(["queued", "running"]);

export function isReportRunActive(status: string): boolean {
  return ACTIVE_RUN_STATUSES.has(status);
}

export function useReportCatalog(enabled = true) {
  return useQuery({
    ...queries.reports.catalog(),
    enabled,
    staleTime: CATALOG_STALE_TIME,
    select: (data) => data.reportCatalog,
  });
}

export function useCannedReports(enabled = true) {
  return useQuery({
    ...queries.reports.canned(),
    enabled,
    staleTime: CATALOG_STALE_TIME,
    select: (data) => data.cannedReports,
  });
}

export function useReportDefinitionList(search: string) {
  return useQuery({
    ...queries.reports.definitionList(search),
    staleTime: 15_000,
    select: (data) =>
      data.reportDefinitions.edges.map((edge) =>
        getFragmentData(ReportDefinitionFieldsFragmentDoc, edge.node),
      ),
  });
}

export function useReportDefinition(id: string | undefined) {
  return useQuery({
    ...queries.reports.definition(id ?? ""),
    enabled: Boolean(id),
    select: (data) => getFragmentData(ReportDefinitionFieldsFragmentDoc, data.reportDefinition),
  });
}

export function useReportRevisions(definitionId: string | undefined, limit?: number) {
  return useQuery({
    ...queries.reports.revisions(definitionId ?? "", limit),
    enabled: Boolean(definitionId),
    select: (data) => data.reportDefinitionRevisions,
  });
}

export function useReportRun(id: string | undefined, options?: { poll?: boolean }) {
  return useQuery({
    ...queries.reports.run(id ?? ""),
    enabled: Boolean(id),
    refetchInterval: options?.poll
      ? (query) => {
          const run = query.state.data
            ? getFragmentData(ReportRunFieldsFragmentDoc, query.state.data.reportRun)
            : undefined;
          return run && isReportRunActive(run.status) ? RUN_POLL_INTERVAL : false;
        }
      : undefined,
    select: (data) => getFragmentData(ReportRunFieldsFragmentDoc, data.reportRun),
  });
}

export function useReportSchedules(definitionId?: string, enabled = true) {
  return useQuery({
    ...queries.reports.schedules(definitionId),
    enabled,
    select: (data) => getFragmentData(ReportScheduleFieldsFragmentDoc, data.reportSchedules),
  });
}

/**
 * `supersede` lets the builder drop its own in-flight preview on the next
 * keystroke. Surfaces that load several previews at once — dashboards — must
 * leave it off, or each tile cancels the one before it.
 */
export function useReportPreview(
  definition: ReportIrInput | null,
  params?: Record<string, unknown>,
  options?: { supersede?: boolean },
) {
  return useQuery({
    ...queries.reports.preview(
      definition ?? { entity: "", columns: [] },
      params,
      options?.supersede ?? false,
    ),
    enabled: Boolean(definition && definition.entity && definition.columns.length > 0),
    staleTime: 30_000,
    retry: false,
    select: (data) => getFragmentData(ReportPreviewFieldsFragmentDoc, data.previewReport),
  });
}

export function useReportDrillThrough(input: ReportDrillInput | null) {
  return useQuery({
    ...queries.reports.drillThrough(
      input ?? { definition: { entity: "", columns: [] }, dimensions: [] },
    ),
    enabled: Boolean(input),
    staleTime: 30_000,
    retry: false,
    select: (data) => getFragmentData(ReportPreviewFieldsFragmentDoc, data.drillThroughReport),
  });
}

export function useReportDashboards(enabled = true) {
  return useQuery({
    ...queries.reports.dashboards(),
    enabled,
    staleTime: 15_000,
    select: (data) =>
      data.reportDashboards.map((dashboard) =>
        getFragmentData(ReportDashboardFieldsFragmentDoc, dashboard),
      ),
  });
}

export function useReportDashboard(id: string | undefined) {
  return useQuery({
    ...queries.reports.dashboard(id ?? ""),
    enabled: Boolean(id),
    select: (data) => getFragmentData(ReportDashboardFieldsFragmentDoc, data.reportDashboard),
  });
}

function useInvalidateDashboards() {
  const queryClient = useQueryClient();

  return async (dashboardId?: string) => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.reports.dashboards().queryKey }),
      dashboardId
        ? queryClient.invalidateQueries({
            queryKey: queries.reports.dashboard(dashboardId).queryKey,
          })
        : Promise.resolve(),
    ]);
  };
}

export function useCreateReportDashboard() {
  const invalidate = useInvalidateDashboards();

  return useMutation({
    mutationFn: createReportDashboard,
    onSuccess: async () => invalidate(),
  });
}

export function useUpdateReportDashboard() {
  const invalidate = useInvalidateDashboards();

  return useMutation({
    mutationFn: updateReportDashboard,
    onSuccess: async (updated) => invalidate(updated.id),
  });
}

export function useDeleteReportDashboard() {
  const invalidate = useInvalidateDashboards();

  return useMutation({
    mutationFn: deleteReportDashboard,
    onSuccess: async () => invalidate(),
  });
}

function useInvalidateDefinitions() {
  const queryClient = useQueryClient();

  return async (definitionId?: string) => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.reports.definitionList._def }),
      definitionId
        ? queryClient.invalidateQueries({
            queryKey: queries.reports.definition(definitionId).queryKey,
          })
        : Promise.resolve(),
      definitionId
        ? queryClient.invalidateQueries({ queryKey: queries.reports.revisions._def })
        : Promise.resolve(),
    ]);
  };
}

export function useCreateReportDefinition() {
  const invalidate = useInvalidateDefinitions();

  return useMutation({
    mutationFn: createReportDefinition,
    onSuccess: async () => invalidate(),
  });
}

export function useUpdateReportDefinition() {
  const invalidate = useInvalidateDefinitions();

  return useMutation({
    mutationFn: updateReportDefinition,
    onSuccess: async (updated) => invalidate(updated.id),
  });
}

export function useDeleteReportDefinition() {
  const invalidate = useInvalidateDefinitions();

  return useMutation({
    mutationFn: deleteReportDefinition,
    onSuccess: async () => invalidate(),
  });
}

export function useForkCannedReport() {
  const invalidate = useInvalidateDefinitions();

  return useMutation({
    mutationFn: forkCannedReport,
    onSuccess: async () => invalidate(),
  });
}

export function useResetCannedFork() {
  const invalidate = useInvalidateDefinitions();

  return useMutation({
    mutationFn: resetCannedFork,
    onSuccess: async (updated) => invalidate(updated.id),
  });
}

export function useRunReport() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: runReport,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: [REPORT_RUN_LIST_QUERY_KEY] });
    },
  });
}

export function useCancelReportRun() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: cancelReportRun,
    onSuccess: async (run) => {
      queryClient.setQueryData(queries.reports.run(run.id).queryKey, { reportRun: run });
      await queryClient.invalidateQueries({ queryKey: [REPORT_RUN_LIST_QUERY_KEY] });
    },
  });
}

function useInvalidateSchedules() {
  const queryClient = useQueryClient();

  return async () => {
    await queryClient.invalidateQueries({
      queryKey: queries.reports.schedules._def,
    });
  };
}

export function useCreateReportSchedule() {
  const invalidate = useInvalidateSchedules();

  return useMutation({
    mutationFn: createReportSchedule,
    onSuccess: async () => invalidate(),
  });
}

export function useUpdateReportSchedule() {
  const invalidate = useInvalidateSchedules();

  return useMutation({
    mutationFn: updateReportSchedule,
    onSuccess: async () => invalidate(),
  });
}

export function useDeleteReportSchedule() {
  const invalidate = useInvalidateSchedules();

  return useMutation({
    mutationFn: deleteReportSchedule,
    onSuccess: async () => invalidate(),
  });
}

export function useReportViews(definitionId?: string, enabled = true) {
  return useQuery({
    ...queries.reports.views(definitionId),
    enabled,
    select: (data) => getFragmentData(ReportViewFieldsFragmentDoc, data.reportViews),
  });
}

function useInvalidateViews() {
  const queryClient = useQueryClient();

  return async () => {
    await queryClient.invalidateQueries({
      queryKey: queries.reports.views._def,
    });
  };
}

export function useCreateReportView() {
  const invalidate = useInvalidateViews();

  return useMutation({
    mutationFn: createReportView,
    onSuccess: async () => invalidate(),
  });
}

export function useUpdateReportView() {
  const invalidate = useInvalidateViews();

  return useMutation({
    mutationFn: updateReportView,
    onSuccess: async () => invalidate(),
  });
}

export function useDeleteReportView() {
  const invalidate = useInvalidateViews();

  return useMutation({
    mutationFn: deleteReportView,
    onSuccess: async () => invalidate(),
  });
}

export function downloadReportRun(run: Pick<ReportRun, "id">): void {
  const link = document.createElement("a");
  link.href = reportRunDownloadUrl(run.id);
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

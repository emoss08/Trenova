import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/** How many portal tasks the history shows at once. */
export const PORTAL_RUNS_PAGE = 25;

/** How many analyses the history shows at once. */
export const ANALYSIS_RUNS_PAGE = 25;

export const agentExtension = createQueryKeys("agentExtension", {
  catalog: () => ({
    queryKey: ["catalog"],
    queryFn: () => apiService.agentExtensionService.getCatalog(),
  }),
  config: (type: string) => ({
    queryKey: ["config", type],
    queryFn: () => apiService.agentExtensionService.getConfig(type),
  }),
  portalSites: () => ({
    queryKey: ["portal-sites"],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentExtensionService.listPortalSites({ signal }),
  }),
  portalRuns: (siteId: string) => ({
    queryKey: ["portal-runs", siteId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentExtensionService.listPortalRuns(
        { siteId: siteId || undefined, limit: PORTAL_RUNS_PAGE },
        { signal },
      ),
  }),
  analysisRuns: () => ({
    queryKey: ["analysis-runs"],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentExtensionService.listAnalysisRuns({ limit: ANALYSIS_RUNS_PAGE }, { signal }),
  }),
  analysisChart: (runId: string, position: number) => ({
    queryKey: ["analysis-chart", runId, position],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentExtensionService.analysisChart(runId, position, { signal }),
  }),
});

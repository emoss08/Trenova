import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/** How many portal tasks the history shows at once. */
export const PORTAL_RUNS_PAGE = 25;

/** How many analyses the history shows at once. */
export const ANALYSIS_RUNS_PAGE = 25;

/** How many web watches the list shows at once. */
export const WEB_WATCHES_PAGE = 50;

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
  portalRun: (runId: string) => ({
    queryKey: ["portal-run", runId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentExtensionService.portalRun(runId, { signal }),
  }),
  analysisRuns: () => ({
    queryKey: ["analysis-runs"],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentExtensionService.listAnalysisRuns({ limit: ANALYSIS_RUNS_PAGE }, { signal }),
  }),
  analysisRun: (runId: string) => ({
    queryKey: ["analysis-run", runId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentExtensionService.analysisRun(runId, { signal }),
  }),
  webWatches: () => ({
    queryKey: ["web-watches"],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentExtensionService.listWebWatches({ limit: WEB_WATCHES_PAGE }, { signal }),
  }),
  webWatchFindings: (watchId: string) => ({
    queryKey: ["web-watch-findings", watchId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentExtensionService.webWatchFindings(watchId, { signal }),
  }),
  analysisChart: (runId: string, position: number) => ({
    queryKey: ["analysis-chart", runId, position],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentExtensionService.analysisChart(runId, position, { signal }),
  }),
});

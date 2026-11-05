import { agentExtensionService } from "@/services/agent-extension";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/** How many portal tasks the history shows at once. */
export const PORTAL_RUNS_PAGE = 25;

/** How many web watches the list shows at once. */
export const WEB_WATCHES_PAGE = 50;

export const agentExtension = createQueryKeys("agentExtension", {
  catalog: () => ({
    queryKey: ["catalog"],
    queryFn: () => agentExtensionService.getCatalog(),
  }),
  config: (type: string) => ({
    queryKey: ["config", type],
    queryFn: () => agentExtensionService.getConfig(type),
  }),
  portalSites: () => ({
    queryKey: ["portal-sites"],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      agentExtensionService.listPortalSites({ signal }),
  }),
  portalRuns: (siteId: string) => ({
    queryKey: ["portal-runs", siteId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      agentExtensionService.listPortalRuns(
        { siteId: siteId || undefined, limit: PORTAL_RUNS_PAGE },
        { signal },
      ),
  }),
  portalRun: (runId: string) => ({
    queryKey: ["portal-run", runId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      agentExtensionService.portalRun(runId, { signal }),
  }),
  webWatches: () => ({
    queryKey: ["web-watches"],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      agentExtensionService.listWebWatches({ limit: WEB_WATCHES_PAGE }, { signal }),
  }),
  webWatchFindings: (watchId: string) => ({
    queryKey: ["web-watch-findings", watchId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      agentExtensionService.webWatchFindings(watchId, { signal }),
  }),
});

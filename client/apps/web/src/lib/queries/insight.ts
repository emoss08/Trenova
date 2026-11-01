import { apiService, type BrowseInsightsParams } from "@/services/api";
import type { InsightSurface } from "@/types/insight";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const insight = createQueryKeys("insight", {
  // The surface is part of the key so a page's slice and the home screen's
  // whole view never share a cache entry.
  active: (limit: number, surface?: InsightSurface) => ({
    queryKey: ["insights-active", limit, surface ?? "all"],
    queryFn: ({ signal }) => apiService.insightService.list(limit, surface, { signal }),
  }),
  detail: (id: string) => ({
    queryKey: ["insight-detail", id],
    queryFn: ({ signal }) => apiService.insightService.detail(id, { signal }),
  }),
  browse: (params: BrowseInsightsParams) => ({
    queryKey: ["insights-browse", params],
    queryFn: ({ signal }) => apiService.insightService.browse(params, { signal }),
  }),
});

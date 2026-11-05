import { insightService, type BrowseInsightsParams } from "@/services/insight";
import type { InsightSurface } from "@/types/insight";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const insight = createQueryKeys("insight", {
  // The surface is part of the key so a page's slice and the home screen's
  // whole view never share a cache entry.
  active: (limit: number, surface?: InsightSurface) => ({
    queryKey: ["insights-active", limit, surface ?? "all"],
    queryFn: () => insightService.list(limit, surface),
  }),
  detail: (id: string) => ({
    queryKey: ["insight-detail", id],
    queryFn: () => insightService.detail(id),
  }),
  browse: (params: BrowseInsightsParams) => ({
    queryKey: ["insights-browse", params],
    queryFn: () => insightService.browse(params),
  }),
});

import { apiService, type BrowseInsightsParams } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const insight = createQueryKeys("insight", {
  active: (limit: number) => ({
    queryKey: ["insights-active", limit],
    queryFn: () => apiService.insightService.list(limit),
  }),
  detail: (id: string) => ({
    queryKey: ["insight-detail", id],
    queryFn: () => apiService.insightService.detail(id),
  }),
  browse: (params: BrowseInsightsParams) => ({
    queryKey: ["insights-browse", params],
    queryFn: () => apiService.insightService.browse(params),
  }),
});

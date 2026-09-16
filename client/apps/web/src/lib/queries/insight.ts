import { apiService, type BrowseInsightsParams } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const insight = createQueryKeys("insight", {
  active: (limit: number) => ({
    queryKey: ["insights-active", limit],
    queryFn: () => apiService.insightService.list(limit),
  }),
  browse: (params: BrowseInsightsParams) => ({
    queryKey: ["insights-browse", params],
    queryFn: () => apiService.insightService.browse(params),
  }),
});

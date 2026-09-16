import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const insight = createQueryKeys("insight", {
  active: (limit: number) => ({
    queryKey: ["insights-active", limit],
    queryFn: () => apiService.insightService.list(limit),
  }),
});

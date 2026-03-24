import { apiService } from "@/services/api";
import type { AnalyticsPage } from "@/types/analytics";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const analytics = createQueryKeys("analytics", {
  get: (page: AnalyticsPage) => ({
    queryKey: ["analytics", page],
    queryFn: async () =>
      apiService.analyticService.get({
        page,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      }),
  }),
});

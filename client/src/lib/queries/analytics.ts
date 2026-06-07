import { getShipmentPageAnalyticsGraphQL } from "@/lib/graphql/shipment";
import { apiService } from "@/services/api";
import type { AnalyticsPage } from "@/types/analytics";

async function getAnalyticsPage(page: AnalyticsPage) {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  if (page === "shipment-management") {
    return getShipmentPageAnalyticsGraphQL({ timezone });
  }
  return apiService.analyticService.get({ page, timezone });
}

export const analytics = {
  get: (page: AnalyticsPage) => ({
    queryKey: ["analytics", page] as const,
    queryFn: () => getAnalyticsPage(page),
  }),
};

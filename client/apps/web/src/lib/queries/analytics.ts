import { getShipmentPageAnalyticsGraphQL } from "@/lib/graphql/shipment";
import { analyticService } from "@/services/analytics";
import type { AnalyticsPage } from "@/types/analytics";

async function getAnalyticsPage(page: AnalyticsPage) {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  if (page === "shipment-management") {
    return getShipmentPageAnalyticsGraphQL({ timezone });
  }
  return analyticService.get({ page, timezone });
}

export const analytics = {
  get: (page: AnalyticsPage) => ({
    queryKey: ["analytics", page] as const,
    queryFn: () => getAnalyticsPage(page),
  }),
};

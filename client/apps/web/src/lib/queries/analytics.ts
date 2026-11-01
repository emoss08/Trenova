import { getApiKeyAnalyticsGraphQL } from "@/lib/graphql/api-key-analytics";
import { getShipmentPageAnalyticsGraphQL } from "@/lib/graphql/shipment";
import type { AnalyticsPage } from "@/types/analytics";

async function getAnalyticsPage(page: AnalyticsPage, signal?: AbortSignal) {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  if (page === "shipment-management") {
    return getShipmentPageAnalyticsGraphQL({ timezone });
  }
  return getApiKeyAnalyticsGraphQL({ timezone, signal });
}

export const analytics = {
  get: (page: AnalyticsPage) => ({
    queryKey: ["analytics", page] as const,
    queryFn: ({ signal }: { signal?: AbortSignal } = {}) => getAnalyticsPage(page, signal),
  }),
};

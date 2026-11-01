import { ApiKeyAnalyticsDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { ApiKeyAnalyticsData } from "@/routes/admin/api-keys/_components/analytics/analytics-data";

export async function getApiKeyAnalyticsGraphQL({
  timezone,
  signal,
}: {
  timezone?: string;
  signal?: AbortSignal;
}): Promise<ApiKeyAnalyticsData> {
  const response = await requestGraphQL({
    document: ApiKeyAnalyticsDocument,
    operationName: "ApiKeyAnalytics",
    variables: { timezone: timezone ?? null },
    signal,
  });

  return response.apiKeyAnalytics;
}

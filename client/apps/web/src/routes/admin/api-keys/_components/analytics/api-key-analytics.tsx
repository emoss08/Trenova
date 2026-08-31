import { analytics } from "@/lib/queries/analytics";
import { useSuspenseQuery } from "@tanstack/react-query";
import { ActiveKeysCard } from "./cards/active-keys-card";
import { RequestsCard } from "./cards/requests-card";
import { RevokedKeysCard } from "./cards/revoked-keys-card";
import { TotalKeysCard } from "./cards/total-keys-card";
import { defaultApiKeyAnalyticsData, type ApiKeyAnalyticsData } from "./analytics-data";

export default function ApiKeyAnalytics() {
  const { data: raw } = useSuspenseQuery(analytics.get("api-key-management"));
  const data = (raw as ApiKeyAnalyticsData) ?? defaultApiKeyAnalyticsData;

  return (
    <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-4 pb-3">
      <TotalKeysCard data={data.totalKeys} />
      <ActiveKeysCard data={data.activeKeys} />
      <RevokedKeysCard data={data.revokedKeys} />
      <RequestsCard data={data.requests30d} />
    </div>
  );
}

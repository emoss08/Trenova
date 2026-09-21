import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { ActivityIcon } from "lucide-react";
import { ActivityItem, EmptyState, OverviewSkeleton } from "./shared";

export type RecentActivity = {
  id: string;
  label: string;
  detail: string;
  status: string;
  occurredAt: number;
};

export function SecurityOverview({
  isLoading,
  providerCount,
  enforcedProviderName,
  directoryStatus,
  activePolicyCount,
  recentActivity,
}: {
  isLoading: boolean;
  providerCount: number;
  enforcedProviderName: string;
  directoryStatus: string;
  activePolicyCount: number;
  recentActivity: RecentActivity[];
}) {
  const t = useT();

  if (isLoading) {
    return <OverviewSkeleton />;
  }

  return (
    <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_360px]">
      <KpiStrip className="self-start">
        <KpiStripItem
          label={t("Providers")}
          value={String(providerCount)}
          sub={providerCount === 1 ? "Enabled provider" : "Enabled providers"}
          tone={providerCount > 0 ? "success" : "muted"}
        />
        <KpiStripItem
          label={t("SSO enforcement")}
          value={enforcedProviderName || "Optional"}
          sub={enforcedProviderName ? "Password fallback restricted" : "Password sign-in allowed"}
          tone={enforcedProviderName ? "warning" : "muted"}
        />
        <KpiStripItem
          label={t("SCIM directory")}
          value={directoryStatus || "Not connected"}
          sub={directoryStatus ? "Provisioning enabled" : "Directory sync inactive"}
          tone={directoryStatus ? "success" : "muted"}
        />
        <KpiStripItem
          label={t("Active policies")}
          value={String(activePolicyCount)}
          sub={activePolicyCount === 1 ? "Policy evaluating" : "Policies evaluating"}
          tone={activePolicyCount > 0 ? "info" : "muted"}
        />
      </KpiStrip>
      <div className="bg-muted/20 rounded-lg border">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <div>
            <div className="text-sm font-medium">{t("Recent security activity")}</div>
            <div className="text-muted-foreground text-xs">
              {t("Latest authentication and risk signals")}
            </div>
          </div>
          <ActivityIcon className="text-muted-foreground size-4" />
        </div>
        <div className="divide-y">
          {recentActivity.length > 0 ? (
            recentActivity.map((activity) => (
              <ActivityItem
                key={activity.id}
                title={t(activity.label)}
                detail={activity.detail}
                badge={activity.status}
                when={formatUnixDateTimeOrDash(activity.occurredAt)}
              />
            ))
          ) : (
            <EmptyState
              icon={<ActivityIcon />}
              label={t("No activity yet")}
              description={t("Sign-in and risk events will appear here after users authenticate.")}
              compact
            />
          )}
        </div>
      </div>
    </div>
  );
}

import { formatUnixDateTimeOrDash } from "@/lib/date";
import {
  ActivityIcon,
  KeyRoundIcon,
  LockKeyholeIcon,
  ShieldCheckIcon,
  UsersRoundIcon,
} from "lucide-react";
import { ActivityItem, EmptyState, OverviewSkeleton, StatusTile } from "./shared";

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
  if (isLoading) {
    return <OverviewSkeleton />;
  }

  return (
    <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatusTile
          icon={<KeyRoundIcon />}
          label="Providers"
          value={String(providerCount)}
          detail={providerCount === 1 ? "Enabled provider" : "Enabled providers"}
          tone={providerCount > 0 ? "active" : "muted"}
        />
        <StatusTile
          icon={<LockKeyholeIcon />}
          label="SSO enforcement"
          value={enforcedProviderName || "Optional"}
          detail={
            enforcedProviderName ? "Password fallback restricted" : "Password sign-in allowed"
          }
          tone={enforcedProviderName ? "warning" : "muted"}
        />
        <StatusTile
          icon={<UsersRoundIcon />}
          label="SCIM directory"
          value={directoryStatus || "Not connected"}
          detail={directoryStatus ? "Provisioning enabled" : "Directory sync inactive"}
          tone={directoryStatus ? "active" : "muted"}
        />
        <StatusTile
          icon={<ShieldCheckIcon />}
          label="Active policies"
          value={String(activePolicyCount)}
          detail={activePolicyCount === 1 ? "Policy evaluating" : "Policies evaluating"}
          tone={activePolicyCount > 0 ? "info" : "muted"}
        />
      </div>
      <div className="rounded-lg border bg-muted/20">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <div>
            <div className="text-sm font-medium">Recent security activity</div>
            <div className="text-xs text-muted-foreground">
              Latest authentication and risk signals
            </div>
          </div>
          <ActivityIcon className="size-4 text-muted-foreground" />
        </div>
        <div className="divide-y">
          {recentActivity.length > 0 ? (
            recentActivity.map((activity) => (
              <ActivityItem
                key={activity.id}
                title={activity.label}
                detail={activity.detail}
                badge={activity.status}
                when={formatUnixDateTimeOrDash(activity.occurredAt)}
              />
            ))
          ) : (
            <EmptyState
              icon={<ActivityIcon />}
              label="No activity yet"
              description="Sign-in and risk events will appear here after users authenticate."
              compact
            />
          )}
        </div>
      </div>
    </div>
  );
}

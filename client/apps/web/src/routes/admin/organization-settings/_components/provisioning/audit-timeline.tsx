import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { toTitleCase } from "@trenova/shared/lib/utils";
import type { ProvisioningAuditRecord } from "@trenova/shared/types/iam";
import { ActivityIcon } from "lucide-react";
import { memo } from "react";
import { ActivityItem, EmptyState, PanelHeader } from "../security-access/shared";

export const AuditTimeline = memo(function AuditTimeline({
  records,
  isLoading,
}: {
  records: ProvisioningAuditRecord[];
  isLoading: boolean;
}) {
  return (
    <div className="rounded-lg border bg-background">
      <PanelHeader
        icon={<ActivityIcon />}
        title="Provisioning audit"
        description="Recent user and group synchronization events."
      />
      <div className="divide-y">
        {isLoading ? (
          <div className="space-y-2 p-3">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : records.length > 0 ? (
          records
            .slice(0, 8)
            .map((record) => (
              <ActivityItem
                key={record.id}
                title={`${toTitleCase(record.action)} ${record.resourceType}`}
                detail={
                  record.errorMessage ||
                  record.externalId ||
                  record.resourceId ||
                  "Provisioning event"
                }
                badge={record.status}
                when={formatUnixDateTimeOrDash(record.createdAt)}
              />
            ))
        ) : (
          <EmptyState
            icon={<ActivityIcon />}
            label="No provisioning events"
            description="SCIM activity will appear after your directory starts syncing."
            compact
          />
        )}
      </div>
    </div>
  );
});

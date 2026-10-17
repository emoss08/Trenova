import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { toTitleCase } from "@trenova/shared/lib/utils";
import type { ProvisioningAuditRecord } from "@trenova/shared/types/iam";
import { ActivityIcon } from "@trenova/shared/components/icons";
import { memo } from "react";
import { ActivityItem, EmptyState, RowSkeleton, SecuritySection } from "../security-access/shared";

const AUDIT_LIMIT = 8;

export const AuditTimeline = memo(function AuditTimeline({
  records,
  isLoading,
}: {
  records: ProvisioningAuditRecord[];
  isLoading: boolean;
}) {
  const t = useT();

  return (
    <SecuritySection
      title={t("Provisioning audit")}
      icon={<ActivityIcon />}
      hint={t("Latest user and group sync events")}
    >
      {isLoading ? (
        <RowSkeleton rows={2} label={t("Loading provisioning events")} />
      ) : records.length > 0 ? (
        <div className="divide-border-subtle divide-y">
          {records.slice(0, AUDIT_LIMIT).map((record) => (
            <ActivityItem
              key={record.id}
              title={t("{0} {1}", toTitleCase(record.action), record.resourceType)}
              detail={
                record.errorMessage ||
                record.externalId ||
                record.resourceId ||
                t("Provisioning event")
              }
              badge={record.status}
              when={formatUnixDateTimeOrDash(record.createdAt)}
            />
          ))}
        </div>
      ) : (
        <EmptyState
          icon={<ActivityIcon />}
          label={t("No provisioning events")}
          description={t("SCIM activity will appear after your directory starts syncing.")}
        />
      )}
    </SecuritySection>
  );
});

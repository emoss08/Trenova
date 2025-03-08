import { formatToUserTimezone } from "@/lib/date";
import { useUser } from "@/stores/user-store";
import { type AuditEntry } from "@/types/audit-entry";
import { AuditEntryActionBadge } from "./audit-column-components";

export default function AuditDetailsHeader({ entry }: { entry?: AuditEntry }) {
  const user = useUser();

  if (!entry) {
    return null;
  }

  const { timestamp, comment, action } = entry;

  return (
    <div className="flex flex-col px-4 pb-2 border-b border-bg-sidebar-border">
      <div className="flex items-center justify-between">
        <h2 className="font-semibold leading-none tracking-tight flex items-center gap-x-2">
          {comment || "-"}
        </h2>
        <AuditEntryActionBadge action={action} />
      </div>
      <p className="text-2xs text-muted-foreground font-normal">
        Entry created on{" "}
        {formatToUserTimezone(timestamp, {
          timezone: user?.timezone,
          timeFormat: user?.timeFormat,
        })}
      </p>
    </div>
  );
}

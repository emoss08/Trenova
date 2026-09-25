import { ExternalLink } from "@/components/link";
import { recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import { accountingSyncNeedsAction, accountingSyncRecordPhase } from "@/lib/accounting-sync";
import type { AccountingSyncObjectState } from "@/lib/graphql/accounting-sync-ledger";
import { queries } from "@/lib/queries";
import type { AccountingSyncRecordStatus } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { BookCheckIcon } from "lucide-react";
import { Link } from "react-router";

type AccountingSyncStateLineProps = {
  objectId: string | null | undefined;
  className?: string;
};

export function AccountingSyncStateLine({ objectId, className }: AccountingSyncStateLineProps) {
  const { allowed } = usePermission(Resource.AccountingSync, Operation.Read);
  const ids = objectId ? [objectId] : [];
  const query = useQuery({
    ...queries.accountingSync.syncObjectStates(ids),
    enabled: allowed && ids.length > 0,
  });
  const state = query.data?.find((item) => item.objectId === objectId);

  if (!allowed || !state) {
    return null;
  }
  return <AccountingSyncStateView state={state} className={className} />;
}

export function AccountingSyncStateView({
  state,
  className,
}: {
  state: AccountingSyncObjectState;
  className?: string;
}) {
  const t = useT();
  const record = state.record;
  const provider = state.providerName;

  const labels: Record<AccountingSyncRecordStatus, string> = {
    Queued: t("Queued for {0}", provider),
    AwaitingApproval: t("Waiting to be released to {0}", provider),
    InFlight: t("Sending to {0}", provider),
    Retrying: t("Retrying {0}", provider),
    Synced: t("In {0}", provider),
    Blocked: t("Held from {0}", provider),
    DeadLettered: t("Not sent to {0}", provider),
    Skipped: t("Not sent to {0}", provider),
    Superseded: t("Replaced by a newer version"),
  };

  const detail = syncDetail(t, state);
  const ledgerLink = recordPath("accounting_sync_record", record.id);

  return (
    <div
      className={cn("flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-sm", className)}
      data-testid="accounting-sync-state"
    >
      <BookCheckIcon aria-hidden className="text-foreground-subtle size-3.5 shrink-0" />
      <Badge variant={phaseTone(accountingSyncRecordPhase(record.status))}>
        {labels[record.status]}
      </Badge>
      {detail ? <span className="text-foreground-muted min-w-0 truncate">{detail}</span> : null}
      {record.status === "Synced" && record.externalUrl ? (
        <ExternalLink href={record.externalUrl} className="text-sm">
          {record.externalDocNumber
            ? t("Open {0} in {1}", record.externalDocNumber, provider)
            : t("Open in {0}", provider)}
        </ExternalLink>
      ) : null}
      <Link
        to={ledgerLink}
        className={cn(
          "text-sm font-medium hover:underline",
          accountingSyncNeedsAction(record.status) ? "text-brand" : "text-foreground-muted",
        )}
      >
        {accountingSyncNeedsAction(record.status)
          ? t("Resolve in the sync ledger")
          : t("Sync history")}
      </Link>
    </div>
  );
}

function syncDetail(t: ReturnType<typeof useT>, state: AccountingSyncObjectState): string {
  const record = state.record;
  switch (record.status) {
    case "Synced":
      return record.syncedAt ? t("since {0}", formatUnixDateTimeShort(record.syncedAt)) : "";
    case "Retrying":
      return record.nextAttemptAt
        ? t("next try {0}", formatUnixDateTimeShort(record.nextAttemptAt))
        : "";
    case "Blocked":
      return record.resolution || record.errorMessage;
    case "DeadLettered":
      return t(
        "{0, plural, one {gave up after # try} other {gave up after # tries}}",
        record.attemptCount,
      );
    case "Skipped":
      return record.skippedReason;
    case "Queued":
    case "AwaitingApproval":
    case "InFlight":
    case "Superseded":
      return "";
  }
}

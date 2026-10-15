import { ExternalLink } from "@/components/link";
import { recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import {
  accountingSyncNeedsAction,
  accountingSyncRecordPhase,
  accountingSyncStateDetail,
  accountingSyncStateLabel,
} from "@/lib/accounting-sync";
import type { AccountingSyncObjectState } from "@/lib/graphql/accounting-sync-ledger";
import { accountingSync } from "@/lib/queries/accounting-sync";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { BookCheckIcon } from "@trenova/shared/components/icons";
import { Link } from "react-router";

type AccountingSyncStateLineProps = {
  objectId: string | null | undefined;
  className?: string;
};

export function AccountingSyncStateLine({ objectId, className }: AccountingSyncStateLineProps) {
  const { allowed } = usePermission(Resource.AccountingSync, Operation.Read);
  const ids = objectId ? [objectId] : [];
  const query = useQuery({
    ...accountingSync.syncObjectStates(ids),
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

  const detail = accountingSyncStateDetail(state, t);
  const ledgerLink = recordPath("accounting_sync_record", record.id);

  return (
    <div
      className={cn("flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-sm", className)}
      data-testid="accounting-sync-state"
    >
      <BookCheckIcon aria-hidden className="text-foreground-subtle size-3.5 shrink-0" />
      <Badge variant={phaseTone(accountingSyncRecordPhase(record.status))}>
        {accountingSyncStateLabel(record.status, provider, t)}
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

import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { SectionPanel } from "@/components/section-panel";
import {
  tableFilterSearchParamsParser,
  tablePaginationSearchParamsParser,
} from "@/hooks/data-table/use-data-table-state";
import { useAccountingSyncActions } from "@/hooks/use-accounting-sync-actions";
import { useAccountingSyncLabels } from "@/hooks/use-accounting-sync-labels";
import { usePermission } from "@/hooks/use-permission";
import { recordPath } from "@/config/record-links";
import type { AccountingSyncSummary } from "@/lib/graphql/accounting-sync-ledger";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryStates } from "nuqs";
import { Link } from "react-router";
import { activeBucket, bucketCount, bucketFilter, type LedgerBucket } from "./ledger-filters";

export function LedgerSummary({ summary }: { summary: AccountingSyncSummary }) {
  const t = useT();
  const [filters, setFilters] = useQueryStates({
    ...tableFilterSearchParamsParser,
    ...tablePaginationSearchParamsParser,
  });
  const current = activeBucket(filters.fieldFilters);

  const select = (bucket: LedgerBucket) => {
    void setFilters({
      fieldFilters: current === bucket ? [] : bucketFilter(bucket),
      pageIndex: 1,
    });
  };

  const attention = bucketCount(summary.counts, "attention");
  const held = bucketCount(summary.counts, "held");

  return (
    <KpiStrip>
      <KpiStripItem
        label={t("Synced")}
        value={bucketCount(summary.counts, "sent")}
        active={current === "sent"}
        onClick={() => select("sent")}
      />
      <KpiStripItem
        label={t("In progress")}
        value={bucketCount(summary.counts, "moving")}
        sub={t("queued, sending or retrying")}
        active={current === "moving"}
        onClick={() => select("moving")}
      />
      <KpiStripItem
        label={t("Waiting for release")}
        value={held}
        tone={held > 0 ? "warning" : undefined}
        active={current === "held"}
        onClick={() => select("held")}
      />
      <KpiStripItem
        label={t("Needs attention")}
        value={attention}
        sub={t("held or failed")}
        tone={attention > 0 ? "danger" : undefined}
        active={current === "attention"}
        onClick={() => select("attention")}
      />
    </KpiStrip>
  );
}

export function LedgerSummarySkeleton() {
  return <KpiStripSkeleton count={4} />;
}

export function LedgerNotices({ summary }: { summary: AccountingSyncSummary }) {
  const t = useT();
  const connection = summary.connection;
  const backfill = summary.activeBackfill;
  const { allowed: canUpdate } = usePermission(Resource.AccountingSync, Operation.Update);
  const { allowed: canManage } = usePermission(Resource.AccountingIntegration, Operation.Manage);
  const actions = useAccountingSyncActions(summary.integrationType, summary.providerName);

  return (
    <>
      {connection?.pausedAt ? (
        <Alert variant="warning" size="sm">
          <AlertTitle>{t("Sending to {0} is paused", summary.providerName)}</AlertTitle>
          <AlertDescription className="flex flex-wrap items-center justify-between gap-2">
            <span>
              {connection.pausedReason
                ? t(
                    "Paused {0} by {1}: {2}",
                    formatUnixDateTimeShort(connection.pausedAt),
                    connection.pausedBy?.name ?? t("someone"),
                    connection.pausedReason,
                  )
                : t(
                    "Paused {0} by {1}. Posted documents keep queueing.",
                    formatUnixDateTimeShort(connection.pausedAt),
                    connection.pausedBy?.name ?? t("someone"),
                  )}
            </span>
            {canUpdate ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                isLoading={actions.resume.isPending}
                onClick={() => actions.resume.mutate()}
              >
                {t("Resume")}
              </Button>
            ) : null}
          </AlertDescription>
        </Alert>
      ) : null}
      {backfill ? (
        <Alert variant={backfill.status === "Failed" ? "destructive" : "info"} size="sm">
          <AlertDescription className="flex flex-wrap items-center justify-between gap-2">
            <span>
              {t(
                "Backfill {0}: documents dated {1} to {2}, {3} queued, {4} already queued.",
                backfill.status === "Paused" ? t("paused") : t("running"),
                formatUnixDateMedium(backfill.rangeStart),
                formatUnixDateMedium(backfill.rangeEnd),
                backfill.enqueuedCount,
                backfill.alreadyQueuedCount,
              )}
            </span>
            {canManage ? (
              <span className="flex gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  isLoading={actions.changeBackfill.isPending}
                  onClick={() =>
                    actions.changeBackfill.mutate({
                      id: backfill.id,
                      action: backfill.status === "Paused" ? "Resume" : "Pause",
                    })
                  }
                >
                  {backfill.status === "Paused" ? t("Resume") : t("Pause")}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={actions.changeBackfill.isPending}
                  onClick={() =>
                    actions.changeBackfill.mutate({ id: backfill.id, action: "Cancel" })
                  }
                >
                  {t("Cancel backfill")}
                </Button>
              </span>
            ) : null}
          </AlertDescription>
        </Alert>
      ) : null}
      <LedgerAttention summary={summary} canUpdate={canUpdate} actions={actions} />
    </>
  );
}

function LedgerAttention({
  summary,
  canUpdate,
  actions,
}: {
  summary: AccountingSyncSummary;
  canUpdate: boolean;
  actions: ReturnType<typeof useAccountingSyncActions>;
}) {
  const t = useT();
  const labels = useAccountingSyncLabels();
  if (summary.attention.length === 0) {
    return null;
  }

  return (
    <SectionPanel
      title={t("Needs attention")}
      count={summary.attention.reduce((total, group) => total + group.count, 0)}
      help={t(
        "Records held or failed for the same reason are grouped, so one fix clears them all. Fix what the reason names, then retry.",
      )}
    >
      <ul className="divide-y">
        {summary.attention.map((group) => (
          <li
            key={`${group.status}:${group.errorCategory ?? ""}:${group.resolution}`}
            className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm"
          >
            <div className="min-w-0 space-y-0.5">
              <p className="truncate">
                {group.resolution ||
                  (group.errorCategory
                    ? labels.errorCategory[group.errorCategory]
                    : labels.status[group.status])}
              </p>
              <p className="text-foreground-muted text-xs">
                {t(
                  "{0, plural, one {# record} other {# records}} since {1}",
                  group.count,
                  formatUnixDateMedium(group.oldestQueuedAt),
                )}{" "}
                ·{" "}
                <Link
                  to={recordPath("accounting_sync_record", group.sampleRecordId)}
                  className="text-brand font-medium hover:underline"
                >
                  {t("Open an example")}
                </Link>
              </p>
            </div>
            {canUpdate && group.errorCategory ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                isLoading={actions.retry.isPending}
                onClick={() => actions.retry.mutate({ errorCategories: [group.errorCategory!] })}
              >
                {t("Retry these")}
              </Button>
            ) : null}
          </li>
        ))}
      </ul>
    </SectionPanel>
  );
}

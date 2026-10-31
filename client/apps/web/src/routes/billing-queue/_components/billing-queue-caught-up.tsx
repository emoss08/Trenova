import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { billingQueue } from "@/lib/queries/billing-queue";
import { CheckIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { formatDurationFromSeconds } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { BillingQueueLane, BillingQueueStats } from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { useBulkRun } from "./use-bulk-run";

/** Lanes that still hold work for a biller; the queue is caught up when they are empty. */
const WORK_LANES: readonly BillingQueueLane[] = [
  "NeedsYou",
  "Ready",
  "Waiting",
  "Checking",
  "Held",
];

export function isCaughtUp(stats: BillingQueueStats | undefined): boolean {
  if (!stats) return false;
  return stats.lanes.every((lane) => !WORK_LANES.includes(lane.lane) || lane.count === 0);
}

/**
 * Nothing left to review: what today came to, and the one thing that may still be
 * waiting, approved drafts to post.
 */
export function BillingQueueCaughtUp({
  stats,
  onShowApproved,
}: {
  stats: BillingQueueStats;
  onShowApproved: () => void;
}) {
  const t = useT();
  const rt = useRichT();
  const { allowed: canPost } = usePermission(Resource.Invoice, Operation.Post);
  const { data: assist } = useQuery(billingQueue.assist());
  const { data: recommendations = [] } = useQuery(billingQueue.recommendations());
  const bulk = useBulkRun();

  const drafts = stats.lanes.find((lane) => lane.lane === "Approved")?.count ?? 0;
  const postable = recommendations.find((r) => r.kind === "PostApproved");
  const pulse = stats.pulse;

  return (
    <div className="flex max-w-105 flex-col items-center gap-3.5 text-center">
      <span
        role="img"
        aria-label={t("Done")}
        className="border-border text-success-foreground flex size-10 items-center justify-center rounded-full border"
      >
        <CheckIcon className="size-4.5" />
      </span>
      <h2 className="text-foreground text-xl font-semibold">{t("You're all caught up")}</h2>
      <p className="text-foreground-muted text-base leading-5">
        {drafts > 0
          ? t(
              "Every shipment that reached billing has been approved. {0, plural, one {# approved draft is} other {# approved drafts are}} still waiting to be posted.",
              drafts,
            )
          : t("Every shipment that reached billing has been approved and posted.")}
      </p>
      <div className="mt-1 flex gap-2">
        {drafts > 0 ? (
          <Button variant="outline" onClick={onShowApproved}>
            {t("See what was approved")}
          </Button>
        ) : null}
        {canPost && postable && postable.count > 0 ? (
          <Button
            isLoading={bulk.isStarting}
            onClick={() => void bulk.start("Post", postable.itemIds)}
          >
            {t("{0, plural, one {Post # draft} other {Post # drafts}}", postable.count)}
          </Button>
        ) : null}
      </div>
      <div className="border-border mt-4.5 flex w-full flex-col gap-2 border-t pt-3.5 text-left">
        {assist?.review ? (
          <div className="flex items-start gap-2.5">
            <NovaBadge size="sm" className="mt-0.5" />
            <p className="text-foreground-muted leading-4.5">
              {rt(
                "<b>Nova</b> reviews each shipment as it reaches billing, so the next ones arrive with its notes.",
                { b: (chunks) => <span className="text-foreground font-medium">{chunks}</span> },
              )}
            </p>
          </div>
        ) : null}
        <div className="text-foreground-muted flex justify-between pl-6 font-mono text-xs">
          <span>
            {t(
              "{0} approved · {1}",
              pulse?.approvedTodayCount ?? 0,
              formatCurrency(Number(pulse?.approvedTodayTotal ?? 0)),
            )}
          </span>
          {pulse?.medianReviewSecondsToday != null ? (
            <span>
              {t("Median review {0}", formatDurationFromSeconds(pulse.medianReviewSecondsToday))}
            </span>
          ) : null}
        </div>
      </div>
    </div>
  );
}

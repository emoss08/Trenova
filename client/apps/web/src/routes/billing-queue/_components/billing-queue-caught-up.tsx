import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { billingQueue } from "@/lib/queries/billing-queue";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
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
 * The queue's lanes as they stand once the work is done: each lane's dot and
 * name with nothing under it, the way the sidebar reads when it has emptied.
 */
const GHOST_LANES: readonly { name: string; total: string }[] = [
  { name: "w-20", total: "w-10" },
  { name: "w-14", total: "w-12" },
  { name: "w-24", total: "w-10" },
  { name: "w-16", total: "w-8" },
];

/**
 * Nothing left to review: the emptied queue, and the one thing that may still
 * be waiting, approved drafts to post.
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
  const showPost = canPost && postable !== undefined && postable.count > 0;

  return (
    <EmptySheet
      className="h-full justify-center"
      sketchClassName="max-w-xs"
      title={t("You're all caught up")}
      description={
        drafts > 0
          ? t(
              "Every shipment that reached billing has been approved. {0, plural, one {# approved draft is} other {# approved drafts are}} still waiting to be posted.",
              drafts,
            )
          : t("Every shipment that reached billing has been approved and posted.")
      }
      action={
        <div className="flex flex-col items-center gap-4">
          {drafts > 0 || showPost ? (
            <div className="flex gap-2">
              {drafts > 0 ? (
                <Button variant="secondary" size="sm" onClick={onShowApproved}>
                  {t("See what was approved")}
                </Button>
              ) : null}
              {showPost ? (
                <Button
                  size="sm"
                  isLoading={bulk.isStarting}
                  onClick={() => void bulk.start("Post", postable.itemIds)}
                >
                  {t("{0, plural, one {Post # draft} other {Post # drafts}}", postable.count)}
                </Button>
              ) : null}
            </div>
          ) : null}
          {assist?.review ? (
            <p className="text-foreground-muted flex max-w-sm items-start gap-2 text-left text-xs leading-relaxed">
              <NovaBadge size="xs" className="mt-0.5 shrink-0" />
              <span>
                {rt(
                  "<b>Nova</b> reviews each shipment as it reaches billing, so the next ones arrive with its notes.",
                  { b: (chunks) => <span className="text-foreground font-medium">{chunks}</span> },
                )}
              </span>
            </p>
          ) : null}
        </div>
      }
      sketch={
        <div className="border-border/70 bg-card divide-border/60 divide-y divide-dashed rounded-md border text-left">
          {GHOST_LANES.map((lane, index) => (
            <div key={index} className="flex items-center gap-2.5 px-3 py-2.5">
              <span className="bg-muted size-1.5 shrink-0 rounded-full" />
              <GhostLine className={`h-2 ${lane.name}`} />
              <GhostLine className={`ml-auto h-2 ${lane.total}`} />
            </div>
          ))}
        </div>
      }
    />
  );
}

import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSingleFlight } from "@/hooks/use-single-flight";
import { usePermission } from "@/hooks/use-permission";
import {
  closePerformanceReview,
  deletePerformanceReview,
  reopenPerformanceReview,
  submitPerformanceReview,
  type PerformanceReviewRow,
} from "@/lib/graphql/performance-review";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { PlusIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  PERFORMANCE_REVIEW_STATUS_LABELS,
  type PerformanceReviewStatus,
} from "@trenova/shared/types/performance-review";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { ListSketch } from "../../employment/list-sketch";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { ReviewsSkeletonBody } from "../development-skeleton";
import { workerReviewsQuery } from "../development-reads";
import { REVIEW_STATUS_VARIANT, ReviewCard, type ReviewPermissions } from "./review-card";
import { reviewCadence } from "./review-cadence";
import { useReviewInvalidation } from "./use-review-invalidation";

type Pending = { id: string; action: "submit" | "reopen" | "close" } | null;

export type ReviewsViewProps = {
  workerId: string;
  workerFirstName: string;
  /** A review a link sent the reader to (`review` in the address, no form). */
  focusId: string | null;
  now: number;
  onStart: () => void;
  onEdit: (review: PerformanceReviewRow) => void;
  onRead: (review: PerformanceReviewRow) => void;
};

/**
 * Development › Reviews: the open review as a panel, beside the cadence the reviews keep and
 * the reviews already closed.
 */
export default function WorkerReviewsTab({
  workerId,
  workerFirstName,
  focusId,
  now,
  onStart,
  onEdit,
  onRead,
}: ReviewsViewProps) {
  const t = useT();
  const { allowed: canCreate } = usePermission(Resource.PerformanceReview, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.PerformanceReview, Operation.Update);
  const { allowed: canSubmit } = usePermission(Resource.PerformanceReview, Operation.Submit);
  const { allowed: canClose } = usePermission(Resource.PerformanceReview, Operation.Close);
  const { allowed: canDelete } = usePermission(Resource.PerformanceReview, Operation.Delete);
  const invalidate = useReviewInvalidation(workerId);
  const [pending, setPending] = useState<Pending>(null);
  const [deleting, setDeleting] = useState<PerformanceReviewRow | null>(null);
  const reviewsQuery = useQuery(workerReviewsQuery(workerId));

  const step = useApiMutation({
    resourceName: "Review",
    scope: { id: `worker-review-step:${workerId}` },
    mutationFn: ({
      review,
      action,
    }: {
      review: PerformanceReviewRow;
      action: "submit" | "reopen" | "close";
    }) => {
      const input = { id: review.id, version: review.version };
      if (action === "submit") return submitPerformanceReview(input);
      if (action === "reopen") return reopenPerformanceReview(input);
      return closePerformanceReview(input);
    },
    onMutate: ({ review, action }) => setPending({ id: review.id, action }),
    onSettled: () => setPending(null),
    onSuccess: () => void invalidate(),
  });
  const remove = useApiMutation({
    resourceName: "Review",
    scope: { id: `worker-review-delete:${workerId}` },
    mutationFn: (review: PerformanceReviewRow) => deletePerformanceReview(review.id),
    onSuccess: () => {
      setDeleting(null);
      void invalidate();
    },
  });

  const stepOnce = useSingleFlight(
    (input: { review: PerformanceReviewRow; action: "submit" | "reopen" | "close" }) =>
      step.mutateAsync(input),
  );
  const removeOnce = useSingleFlight((review: PerformanceReviewRow) => remove.mutateAsync(review));

  if (reviewsQuery.isPending) return <ReviewsSkeletonBody />;
  if (reviewsQuery.isError) {
    return (
      <ErrorState
        layout="sheet"
        error={reviewsQuery.error}
        onRetry={() => void reviewsQuery.refetch()}
        title={t("Couldn't load the reviews")}
        sketch={<ListSketch />}
      />
    );
  }

  const reviews = reviewsQuery.data;
  const open = reviews.filter((review) => review.status !== "Closed");
  const closed = reviews
    .filter((review) => review.status === "Closed")
    .sort((a, b) => (b.closedAt ?? b.periodEnd) - (a.closedAt ?? a.periodEnd));
  const cadence = reviewCadence(reviews, now);
  const permissions: ReviewPermissions = { canUpdate, canSubmit, canClose, canDelete };
  const run = (review: PerformanceReviewRow, action: "submit" | "reopen" | "close") => {
    void stepOnce.run({ review, action });
  };

  return (
    <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_24rem]">
      <div className="flex min-w-0 flex-col gap-4">
        {open.length === 0 ? (
          <SectionPanel title={t("Open review")}>
            <EmptySheet
              title={t("No review open")}
              description={
                cadence.nextDueAt
                  ? t("The next review is due {0}.", formatUnixDateMedium(cadence.nextDueAt))
                  : t(
                      "Start a review to rate the period, set goals and send it to the worker to sign.",
                    )
              }
              sketch={<ListSketch />}
              action={
                canCreate ? (
                  <Button size="sm" onClick={onStart}>
                    <PlusIcon className="size-3.5" />
                    {t("Start a review")}
                  </Button>
                ) : undefined
              }
            />
          </SectionPanel>
        ) : (
          open.map((review) => (
            <ReviewCard
              key={review.id}
              review={review}
              workerFirstName={workerFirstName}
              permissions={permissions}
              pending={pending?.id === review.id ? pending.action : null}
              highlighted={focusId === review.id}
              onEdit={onEdit}
              onSubmit={(target) => run(target, "submit")}
              onReopen={(target) => run(target, "reopen")}
              onClose={(target) => run(target, "close")}
              onDelete={setDeleting}
            />
          ))
        )}
      </div>

      <div className="flex min-w-0 flex-col gap-4">
        <SectionPanel
          title={t("Cadence")}
          hint={
            cadence.templateName && cadence.cadenceMonths
              ? t(
                  "{0} · every {1, plural, one {# month} other {# months}}",
                  cadence.templateName,
                  cadence.cadenceMonths,
                )
              : (cadence.templateName ?? undefined)
          }
          action={
            canCreate && open.length > 0 ? (
              <Button size="sm" variant="secondary" onClick={onStart}>
                {t("Start a review")}
              </Button>
            ) : null
          }
        >
          <div className="px-3 py-1.5">
            <DescriptionList layout="split">
              <DescriptionItem label={t("Next review due")} numeric>
                {cadence.nextDueAt ? (
                  <>
                    {formatUnixDateMedium(cadence.nextDueAt)}
                    {cadence.daysUntilDue !== null ? (
                      <span
                        className={
                          cadence.daysUntilDue < 0
                            ? "text-danger-foreground"
                            : "text-muted-foreground"
                        }
                      >
                        {" · "}
                        {cadence.daysUntilDue < 0
                          ? t(
                              "{0, plural, one {# day} other {# days}} overdue",
                              -cadence.daysUntilDue,
                            )
                          : t("in {0, plural, one {# day} other {# days}}", cadence.daysUntilDue)}
                      </span>
                    ) : null}
                  </>
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Last reviewed")} numeric>
                {cadence.lastReviewedAt ? (
                  formatUnixDateMedium(cadence.lastReviewedAt)
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
            </DescriptionList>
          </div>
        </SectionPanel>

        <SectionPanel title={t("History")} count={closed.length}>
          {closed.length === 0 ? (
            <SectionPanelQuiet>{t("No closed reviews yet.")}</SectionPanelQuiet>
          ) : (
            <ScrollArea className="max-h-[24rem]" maskVariant="card">
              <HoverGlideGroup className="flex flex-col gap-px p-1.5">
                {closed.map((review) => {
                  const status = review.status as PerformanceReviewStatus;
                  const signed = review.acknowledgedAt != null;
                  return (
                    <button
                      key={review.id}
                      type="button"
                      data-glide-target
                      data-testid={`review-history-${review.id}`}
                      onClick={() => onRead(review)}
                      className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-md ui-focus-ring px-2 py-2 text-left"
                    >
                      <span className="min-w-0">
                        <span className="block truncate text-sm font-medium">{review.title}</span>
                        <span className="block truncate text-xs text-muted-foreground tabular-nums">
                          {historyLine(review, t)}
                        </span>
                      </span>
                      <span className="flex items-center gap-3">
                        <Badge variant={signed ? "success" : REVIEW_STATUS_VARIANT[status]} dot>
                          {signed ? t("Signed") : PERFORMANCE_REVIEW_STATUS_LABELS[status]}
                        </Badge>
                        <span className="w-7 text-right text-sm font-medium tabular-nums">
                          {review.overallScore ? Number(review.overallScore).toFixed(1) : "—"}
                        </span>
                      </span>
                    </button>
                  );
                })}
              </HoverGlideGroup>
            </ScrollArea>
          )}
        </SectionPanel>
      </div>

      <ConfirmDialog
        open={deleting !== null}
        title={deleting ? t("Delete the {0}?", deleting.title) : t("Delete this draft?")}
        description={t(
          "The draft, its ratings and its goals are deleted. {0} never saw it. This cannot be undone.",
          workerFirstName,
        )}
        confirmLabel={t("Delete draft")}
        pending={removeOnce.pending}
        onConfirm={() => {
          if (deleting) void removeOnce.run(deleting);
        }}
        onOpenChange={(isOpen) => {
          if (!isOpen) setDeleting(null);
        }}
      />
    </div>
  );
}

function historyLine(review: PerformanceReviewRow, t: ReturnType<typeof useT>): string {
  const who = review.reviewer?.name;
  if (review.acknowledgedAt) {
    const on = formatUnixDateMedium(review.acknowledgedAt);
    return who ? t("Signed {0} · {1}", on, who) : t("Signed {0}", on);
  }
  const on = formatUnixDateMedium(review.closedAt ?? review.periodEnd);
  return who ? t("Closed {0} · {1}", on, who) : t("Closed {0}", on);
}

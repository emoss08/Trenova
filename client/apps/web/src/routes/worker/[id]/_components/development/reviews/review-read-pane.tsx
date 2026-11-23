import { WorkPane, WorkPaneBody, WorkPaneCancel, WorkPaneFooter } from "@/components/work-pane";
import type { PerformanceReviewRow } from "@/lib/graphql/performance-review";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  PERFORMANCE_REVIEW_STATUS_LABELS,
  type PerformanceReviewStatus,
} from "@trenova/shared/types/performance-review";
import { useState } from "react";
import { OverallScore, REVIEW_STATUS_VARIANT, ReviewBody, reviewMeta } from "./review-card";

/** A past review read in the pane, as the worker signed it. */
export function ReviewReadPane({
  review: requested,
  onClose,
}: {
  review: PerformanceReviewRow | null;
  onClose: () => void;
}) {
  const t = useT();
  const [held, setHeld] = useState(requested);
  if (requested !== null && requested !== held) setHeld(requested);
  const review = requested ?? held;
  const status = review?.status as PerformanceReviewStatus | undefined;
  const score = review?.overallScore ? Number(review.overallScore) : null;

  return (
    <WorkPane
      open={requested !== null}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={review?.title ?? t("Review")}
      contentKey={review?.id}
    >
      <WorkPaneBody className="px-0">
        {review && status ? (
          <>
            <div className="flex flex-col gap-1.5 px-4">
              <div className="flex items-center gap-2">
                <Badge variant={REVIEW_STATUS_VARIANT[status] ?? "neutral"} dot>
                  {PERFORMANCE_REVIEW_STATUS_LABELS[status] ?? t("Review")}
                </Badge>
                <OverallScore score={score} className="ml-auto" />
              </div>
              <p className="text-xs text-muted-foreground tabular-nums">{reviewMeta(review)}</p>
              {review.acknowledgedAt ? (
                <p className="text-xs text-muted-foreground">
                  {t(
                    "Signed by the worker on {0} in Dash.",
                    formatUnixDateMedium(review.acknowledgedAt),
                  )}
                </p>
              ) : null}
              {review.closedAt ? (
                <p className="text-xs text-muted-foreground">
                  {t("Closed {0}.", formatUnixDateMedium(review.closedAt))}
                </p>
              ) : null}
            </div>
            <div className="@container/review">
              <ReviewBody review={review} narrow />
            </div>
          </>
        ) : null}
      </WorkPaneBody>
      <WorkPaneFooter>
        <WorkPaneCancel>{t("Close")}</WorkPaneCancel>
      </WorkPaneFooter>
    </WorkPane>
  );
}

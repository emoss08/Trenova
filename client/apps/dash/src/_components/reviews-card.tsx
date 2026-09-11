import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  acknowledgeMyReview,
  fetchMyReviews,
  type PortalReview,
} from "@trenova/shared/lib/graphql/driver-portal";
import { cn } from "@trenova/shared/lib/utils";
import {
  PERFORMANCE_REVIEW_STATUS_LABELS,
  REVIEW_GOAL_STATUS_LABELS,
  REVIEW_SCORE_LABELS,
  type PerformanceReviewStatus,
  type ReviewGoalStatus,
} from "@trenova/shared/types/performance-review";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ClipboardCheckIcon, PenLineIcon, TargetIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

export const DASH_REVIEWS_KEY = "dash-reviews";

export function ReviewsCard() {
  const t = useT();

  const reviews = useQuery({
    queryKey: [DASH_REVIEWS_KEY],
    queryFn: ({ signal }) => fetchMyReviews({ signal }),
  });

  if (reviews.isPending) {
    return <Skeleton className="h-40 w-full rounded-2xl" />;
  }
  if (!reviews.data) {
    return null;
  }

  const waiting = reviews.data.filter((review) => review.status === "Submitted").length;

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <ClipboardCheckIcon className="size-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold">{t("Reviews")}</h2>
        </div>
        {waiting > 0 ? <Badge variant="warning">{t("{0} waiting for you", waiting)}</Badge> : null}
      </div>

      {reviews.data.length === 0 ? (
        <p className="mt-3 text-xs text-muted-foreground">
          {t("No reviews yet. Your manager will share one here when it is ready.")}
        </p>
      ) : (
        <ul className="mt-3 flex flex-col gap-3">
          {reviews.data.map((review) => (
            <ReviewRow key={review.id} review={review} />
          ))}
        </ul>
      )}
    </div>
  );
}

function ReviewRow({ review }: { review: PortalReview }) {
  const t = useT();

  const queryClient = useQueryClient();
  const [comment, setComment] = useState("");
  const status = review.status as PerformanceReviewStatus;
  const needsSignOff = status === "Submitted";

  const acknowledge = useMutation({
    mutationFn: () => acknowledgeMyReview(review.id, comment || undefined),
    onSuccess: async () => {
      toast.success(t("Signed — thanks. Your manager can see your comment."));
      await queryClient.invalidateQueries({ queryKey: [DASH_REVIEWS_KEY] });
    },
    onError: (error: Error) => {
      toast.error(error.message || "Could not sign off. Try again.");
    },
  });

  return (
    <li data-testid={`dash-review-${review.id}`} className="rounded-xl border border-border p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-sm font-medium">{t(review.title)}</p>
          <p className="text-[11px] text-muted-foreground">
            {formatUnixDate(review.periodStart)} – {formatUnixDate(review.periodEnd)}
            {review.reviewer?.name ? ` · ${review.reviewer.name}` : ""}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {review.overallScore ? (
            <span className="text-lg font-semibold tabular-nums">{review.overallScore}</span>
          ) : null}
          <Badge variant={needsSignOff ? "warning" : "outline"}>
            {PERFORMANCE_REVIEW_STATUS_LABELS[status] ?? status}
          </Badge>
        </div>
      </div>

      {review.summary ? <p className="mt-2 text-sm">{review.summary}</p> : null}

      <ul className="mt-2 flex flex-col gap-1">
        {review.ratings.map((rating) => (
          <li key={rating.key} className="flex items-center gap-2 text-xs">
            <span className="min-w-32 truncate">{t(rating.label)}</span>
            <span className="flex items-center gap-0.5" aria-hidden>
              {[1, 2, 3, 4, 5].map((mark) => (
                <span
                  key={mark}
                  className={cn(
                    "size-1.5 rounded-full",
                    rating.score != null && mark <= rating.score
                      ? "bg-primary"
                      : "bg-muted-foreground/25",
                  )}
                />
              ))}
            </span>
            <span className="text-muted-foreground">
              {rating.score == null ? "—" : REVIEW_SCORE_LABELS[rating.score]}
            </span>
          </li>
        ))}
      </ul>

      {review.goals.length > 0 ? (
        <ul className="mt-2 flex flex-col gap-1">
          {review.goals.map((goal) => (
            <li key={goal.id} className="flex items-center gap-1.5 text-xs">
              <TargetIcon className="size-3.5 text-muted-foreground" />
              <span>{t(goal.title)}</span>
              <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
                {REVIEW_GOAL_STATUS_LABELS[goal.status as ReviewGoalStatus] ?? goal.status}
              </Badge>
            </li>
          ))}
        </ul>
      ) : null}

      {review.workerComment ? (
        <p className="mt-2 rounded-md bg-muted/40 px-2 py-1 text-xs">{review.workerComment}</p>
      ) : null}

      {needsSignOff ? (
        <div className="mt-3 flex flex-col gap-1.5 border-t border-border pt-3">
          <label className="text-[11px] text-muted-foreground" htmlFor={`comment-${review.id}`}>
            {t("Your comment")}
          </label>
          <textarea
            id={`comment-${review.id}`}
            className="min-h-16 rounded-md border border-border bg-background p-2 text-xs"
            placeholder={t("Optional — anything you want your manager to see")}
            value={comment}
            onChange={(event) => setComment(event.target.value)}
          />
          <Button
            size="sm"
            className="h-8 self-start"
            disabled={acknowledge.isPending}
            onClick={() => acknowledge.mutate()}
          >
            <PenLineIcon className="size-3.5" />
            {acknowledge.isPending ? "Signing…" : "Sign off"}
          </Button>
        </div>
      ) : null}
    </li>
  );
}

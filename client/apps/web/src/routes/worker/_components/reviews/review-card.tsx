import { useT } from "@trenova/shared/i18n/use-t";
import type { PerformanceReviewRow } from "@/lib/graphql/performance-review";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  PERFORMANCE_REVIEW_STATUS_HINTS,
  PERFORMANCE_REVIEW_STATUS_LABELS,
  REVIEW_GOAL_STATUS_LABELS,
  REVIEW_SCORE_LABELS,
  type PerformanceReviewStatus,
  type ReviewGoalStatus,
} from "@trenova/shared/types/performance-review";
import {
  CheckCircle2Icon,
  PencilIcon,
  RotateCcwIcon,
  SendIcon,
  TargetIcon,
  Trash2Icon,
} from "lucide-react";

export type ReviewPermissions = {
  canUpdate: boolean;
  canSubmit: boolean;
  canClose: boolean;
  canDelete: boolean;
};

type ReviewCardProps = {
  review: PerformanceReviewRow;
  permissions: ReviewPermissions;
  busy: boolean;
  onEdit: (review: PerformanceReviewRow) => void;
  onSubmit: (review: PerformanceReviewRow) => void;
  onReopen: (review: PerformanceReviewRow) => void;
  onClose: (review: PerformanceReviewRow) => void;
  onDelete: (review: PerformanceReviewRow) => void;
};

const STATUS_VARIANT: Record<
  PerformanceReviewStatus,
  "active" | "warning" | "inactive" | "outline"
> = {
  Draft: "outline",
  Submitted: "warning",
  Acknowledged: "active",
  Closed: "outline",
};

export function ReviewCard({
  review,
  permissions,
  busy,
  onEdit,
  onSubmit,
  onReopen,
  onClose,
  onDelete,
}: ReviewCardProps) {
  const t = useT();

  const status = review.status as PerformanceReviewStatus;
  const isDraft = status === "Draft";
  const isSubmitted = status === "Submitted";
  const isClosed = status === "Closed";

  return (
    <div
      data-testid={`review-${review.id}`}
      className={cn("bg-card flex flex-col gap-3 rounded-xl border p-4", !isClosed && "shadow-xs")}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-semibold">{t(review.title)}</p>
          <p className="text-muted-foreground text-[11px]">
            {formatUnixDate(review.periodStart)} – {formatUnixDate(review.periodEnd)}
            {review.reviewer?.name ? ` · ${review.reviewer.name}` : ""}
            {review.template?.name ? ` · ${review.template.name}` : ""}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {review.overallScore ? (
            <span className="text-lg font-semibold tabular-nums">{review.overallScore}</span>
          ) : null}
          <Badge variant={STATUS_VARIANT[status] ?? "outline"}>
            {PERFORMANCE_REVIEW_STATUS_LABELS[status] ?? status}
          </Badge>
        </div>
      </div>

      <p className="text-muted-foreground text-xs">
        {PERFORMANCE_REVIEW_STATUS_HINTS[status] ?? ""}
      </p>

      {review.ratings.length > 0 ? (
        <ul className="flex flex-col gap-1.5">
          {review.ratings.map((rating) => (
            <li key={rating.key} className="flex items-center gap-2 text-xs">
              <span className="min-w-40 truncate">{t(rating.label)}</span>
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
              <span className="text-muted-foreground tabular-nums">
                {rating.score == null
                  ? t("Not rated")
                  : `${rating.score} — ${REVIEW_SCORE_LABELS[rating.score] ?? ""}`}
              </span>
              {rating.comment ? (
                <span className="text-muted-foreground truncate">· {rating.comment}</span>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}

      {review.summary ? <p className="text-sm">{review.summary}</p> : null}

      {review.goals.length > 0 ? (
        <ul className="flex flex-col gap-1">
          {review.goals.map((goal) => (
            <li key={goal.id} className="flex items-center gap-1.5 text-xs">
              <TargetIcon className="text-muted-foreground size-3.5" />
              <span>{t(goal.title)}</span>
              <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
                {REVIEW_GOAL_STATUS_LABELS[goal.status as ReviewGoalStatus] ?? goal.status}
              </Badge>
              {goal.dueAt ? (
                <span className="text-muted-foreground">{t("by {0}", formatUnixDate(goal.dueAt))}</span>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}

      {review.workerComment ? (
        <p className="bg-muted/40 rounded-md px-2.5 py-1.5 text-xs">
          <span className="font-medium">{t("The worker replied:")} </span>
          {review.workerComment}
        </p>
      ) : null}

      {isClosed && review.nextReviewAt ? (
        <p className="text-muted-foreground text-[11px]">
          {t("Next review due {0}.", formatUnixDate(review.nextReviewAt))}
        </p>
      ) : null}

      <div className="flex flex-wrap items-center gap-2">
        {isDraft && permissions.canUpdate ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onEdit(review)}>
            <PencilIcon className="size-3.5" />
            {t("Edit")}
          </Button>
        ) : null}
        {isDraft && permissions.canSubmit ? (
          <Button size="sm" disabled={busy} onClick={() => onSubmit(review)}>
            <SendIcon className="size-3.5" />
            {t("Submit")}
          </Button>
        ) : null}
        {isSubmitted && permissions.canClose ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onReopen(review)}>
            <RotateCcwIcon className="size-3.5" />
            {t("Reopen")}
          </Button>
        ) : null}
        {(isSubmitted || status === "Acknowledged") && permissions.canClose ? (
          <Button size="sm" disabled={busy} onClick={() => onClose(review)}>
            <CheckCircle2Icon className="size-3.5" />
            {t("Close review")}
          </Button>
        ) : null}
        {isDraft && permissions.canDelete ? (
          <Button
            size="sm"
            variant="ghost"
            className="text-destructive ml-auto"
            disabled={busy}
            aria-label={`Delete ${review.title}`}
            onClick={() => onDelete(review)}
          >
            <Trash2Icon className="size-3.5" />
          </Button>
        ) : null}
      </div>
    </div>
  );
}

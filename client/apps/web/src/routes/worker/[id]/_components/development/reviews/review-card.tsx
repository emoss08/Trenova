import type { PerformanceReviewRow } from "@/lib/graphql/performance-review";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  PERFORMANCE_REVIEW_STATUS_HINTS,
  PERFORMANCE_REVIEW_STATUS_LABELS,
  REVIEW_GOAL_STATUS_LABELS,
  type PerformanceReviewStatus,
  type ReviewGoalStatus,
} from "@trenova/shared/types/performance-review";
import { useEffect, useRef } from "react";
import { RatingScale, scoreWord } from "./rating-scale";

export type ReviewPermissions = {
  canUpdate: boolean;
  canSubmit: boolean;
  canClose: boolean;
  canDelete: boolean;
};

export const REVIEW_STATUS_VARIANT: Record<
  PerformanceReviewStatus,
  "success" | "warning" | "neutral" | "info"
> = {
  Draft: "neutral",
  Submitted: "warning",
  Acknowledged: "success",
  Closed: "neutral",
};

const GOAL_VARIANT: Record<ReviewGoalStatus, "neutral" | "success" | "info"> = {
  Open: "info",
  Done: "success",
  Dropped: "neutral",
};

/** "Jan 1 to Dec 31, 2026 · reviewer Dana Whitfield · Driver annual" */
export function reviewMeta(review: PerformanceReviewRow): string {
  const period = translate(
    "{0} to {1}",
    formatUnixDateMedium(review.periodStart),
    formatUnixDateMedium(review.periodEnd),
  );
  const parts = [period];
  if (review.reviewer?.name) parts.push(translate("reviewer {0}", review.reviewer.name));
  if (review.template?.name) parts.push(review.template.name);
  return parts.join(" · ");
}

/** The overall score as a figure with its word; nothing while no criterion is rated. */
export function OverallScore({ score, className }: { score: number | null; className?: string }) {
  const t = useT();
  return (
    <div className={cn("flex items-baseline gap-2", className)}>
      <span className="text-3xl leading-none font-semibold tracking-tight tabular-nums">
        {score == null ? "—" : score.toFixed(1)}
      </span>
      <span className="text-sm font-medium">
        {score == null ? t("Not rated yet") : scoreWord(score)}
      </span>
    </div>
  );
}

/** The ratings, summary, strengths, goals and the worker's reply, as the worker reads them. */
export function ReviewBody({
  review,
  narrow = false,
}: {
  review: PerformanceReviewRow;
  narrow?: boolean;
}) {
  const t = useT();
  return (
    <>
      {review.ratings.length > 0 ? (
        <ul className="py-1">
          {review.ratings.map((rating) => {
            const word = scoreWord(rating.score);
            return (
              <li
                key={rating.key}
                className={cn(
                  "grid items-center gap-x-4 gap-y-1 border-border-subtle px-4 py-2 not-first:border-t",
                  narrow
                    ? "grid-cols-[minmax(0,1fr)_auto_5.5rem]"
                    : "grid-cols-[minmax(0,1fr)_auto_5.5rem] @2xl/review:grid-cols-[11rem_auto_6rem_minmax(0,1fr)]",
                )}
              >
                <span className="text-sm font-medium">{rating.label}</span>
                <RatingScale score={rating.score} label={rating.label} />
                <span
                  className={cn(
                    "text-right text-xs whitespace-nowrap @2xl/review:text-left",
                    word ? "text-foreground" : "text-foreground-subtle",
                    narrow && "@2xl/review:text-right",
                  )}
                >
                  {word ?? t("Not rated")}
                </span>
                {rating.comment ? (
                  <span
                    className={cn(
                      "col-span-full text-xs text-muted-foreground",
                      !narrow && "@2xl/review:col-span-1",
                    )}
                  >
                    {rating.comment}
                  </span>
                ) : null}
              </li>
            );
          })}
        </ul>
      ) : null}
      <TextBlock title={t("Summary")} text={review.summary} />
      <TextBlock title={t("Strengths")} text={review.strengths} />
      <TextBlock title={t("Where to improve")} text={review.improvements} />
      {review.goals.length > 0 ? (
        <div className="border-t border-border-subtle px-4 py-3">
          <h4 className="mb-1.5 text-xs font-medium text-foreground-subtle">{t("Goals")}</h4>
          <ul>
            {review.goals.map((goal) => (
              <li
                key={goal.id}
                className="grid grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-3 border-border-subtle py-1.5 not-first:border-t"
              >
                <span className="text-sm">{goal.title}</span>
                <span className="text-xs whitespace-nowrap text-muted-foreground tabular-nums">
                  {goal.dueAt ? t("by {0}", formatUnixDateMedium(goal.dueAt)) : null}
                </span>
                <Badge variant={GOAL_VARIANT[goal.status as ReviewGoalStatus] ?? "neutral"} dot>
                  {REVIEW_GOAL_STATUS_LABELS[goal.status as ReviewGoalStatus] ?? t("Goal")}
                </Badge>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      {review.workerComment ? (
        <TextBlock title={t("The worker replied")} text={review.workerComment} />
      ) : null}
    </>
  );
}

function TextBlock({ title, text }: { title: string; text: string | null | undefined }) {
  if (!text) return null;
  return (
    <div className="border-t border-border-subtle px-4 py-3">
      <h4 className="mb-1.5 text-xs font-medium text-foreground-subtle">{title}</h4>
      <p className="max-w-prose text-sm text-pretty whitespace-pre-wrap">{text}</p>
    </div>
  );
}

type ReviewCardProps = {
  review: PerformanceReviewRow;
  workerFirstName: string;
  permissions: ReviewPermissions;
  /** The action in flight on this review, so its button holds. */
  pending: "submit" | "reopen" | "close" | null;
  highlighted?: boolean;
  onEdit: (review: PerformanceReviewRow) => void;
  onSubmit: (review: PerformanceReviewRow) => void;
  onReopen: (review: PerformanceReviewRow) => void;
  onClose: (review: PerformanceReviewRow) => void;
  onDelete: (review: PerformanceReviewRow) => void;
};

/**
 * The open review as a panel: who reviews what period, the overall score, each criterion on
 * the five-step scale, the summary and goals, and the step it is waiting on in the footer.
 */
export function ReviewCard({
  review,
  workerFirstName,
  permissions,
  pending,
  highlighted = false,
  onEdit,
  onSubmit,
  onReopen,
  onClose,
  onDelete,
}: ReviewCardProps) {
  const t = useT();
  const ref = useRef<HTMLElement>(null);
  const status = review.status as PerformanceReviewStatus;
  const isDraft = status === "Draft";
  const isSubmitted = status === "Submitted";
  const busy = pending !== null;
  const score = review.overallScore ? Number(review.overallScore) : null;

  useEffect(() => {
    if (highlighted) ref.current?.scrollIntoView({ block: "nearest" });
  }, [highlighted]);

  const footerLeft = [
    isDraft && permissions.canDelete ? (
      <Button
        key="delete"
        size="sm"
        variant="outline"
        disabled={busy}
        onClick={() => onDelete(review)}
      >
        {t("Delete")}
      </Button>
    ) : null,
    isDraft && permissions.canUpdate ? (
      <Button key="edit" size="sm" variant="outline" disabled={busy} onClick={() => onEdit(review)}>
        {t("Edit")}
      </Button>
    ) : null,
    isSubmitted && permissions.canClose ? (
      <Button
        key="reopen"
        size="sm"
        variant="outline"
        disabled={busy && pending !== "reopen"}
        isLoading={pending === "reopen"}
        onClick={() => onReopen(review)}
      >
        {t("Reopen")}
      </Button>
    ) : null,
  ].filter(Boolean);
  const primary =
    isDraft && permissions.canSubmit ? (
      <Button
        size="sm"
        disabled={busy && pending !== "submit"}
        isLoading={pending === "submit"}
        onClick={() => onSubmit(review)}
      >
        {t("Submit to {0}", workerFirstName)}
      </Button>
    ) : (isSubmitted || status === "Acknowledged") && permissions.canClose ? (
      <Button
        size="sm"
        disabled={busy && pending !== "close"}
        isLoading={pending === "close"}
        onClick={() => onClose(review)}
      >
        {t("Close review")}
      </Button>
    ) : null;

  return (
    <section
      ref={ref}
      aria-label={review.title}
      data-testid={`review-${review.id}`}
      className={cn(
        "@container/review flex flex-col overflow-hidden rounded-lg border bg-card",
        highlighted && "animate-attention-a",
      )}
    >
      <div className="grid grid-cols-1 items-end gap-4 border-b border-border-subtle px-4 pt-3 pb-3.5 @lg/review:grid-cols-[minmax(0,1fr)_auto]">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-sm font-semibold">{review.title}</h3>
            <Badge variant={REVIEW_STATUS_VARIANT[status] ?? "neutral"} dot>
              {PERFORMANCE_REVIEW_STATUS_LABELS[status] ?? t("Review")}
            </Badge>
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">{reviewMeta(review)}</p>
          <p className="mt-1.5 text-xs text-muted-foreground">
            {PERFORMANCE_REVIEW_STATUS_HINTS[status] ?? ""}
          </p>
        </div>
        <div className="flex flex-col @lg/review:items-end">
          <OverallScore score={score} />
          <span className="text-xs text-muted-foreground">{t("Overall")}</span>
        </div>
      </div>
      <ReviewBody review={review} />
      {footerLeft.length > 0 || primary ? (
        <div className="flex flex-wrap items-center justify-between gap-2 border-t px-3 py-2">
          <div className="flex items-center gap-2">{footerLeft}</div>
          {primary}
        </div>
      ) : null}
    </section>
  );
}

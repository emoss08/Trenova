import { ClockRewindIcon } from "@trenova/shared/components/icons";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatDistanceToNow } from "date-fns";
import { useMemo } from "react";
import {
  describeReviewDecision,
  groupReviewRounds,
  type ReviewEntryData,
  type ReviewRound,
} from "./review-rounds";

const TONE_VARIANTS: Record<ReturnType<typeof describeReviewDecision>["tone"], BadgeVariant> = {
  neutral: "accent-sky",
  positive: "success",
  negative: "danger",
  warning: "warning",
  muted: "neutral",
};

function actorName(review: ReviewEntryData, t: TranslateFn): string {
  if (review.actor?.name) return review.actor.name;
  if (review.decision === "Expired") return t("System");
  return t("Someone");
}

function ReviewEntry({ review }: { review: ReviewEntryData }) {
  const t = useT();
  const decision = describeReviewDecision(review.decision);
  return (
    <li className="flex items-start justify-between gap-3 px-3 py-1.5 text-xs">
      <div className="min-w-0 space-y-0.5">
        <div className="flex items-center gap-1.5">
          <Badge variant={TONE_VARIANTS[decision.tone]} className="text-2xs px-1 py-0">
            {t(decision.label)}
          </Badge>
          <span className="font-medium">{actorName(review, t)}</span>
          {review.requestedReviewer?.name && (
            <span className="text-muted-foreground">
              {t("asked {0} to review", review.requestedReviewer.name)}
            </span>
          )}
        </div>
        {review.comment && <p className="text-muted-foreground">{review.comment}</p>}
        {review.reviewerNote && (
          <p className="bg-sunken rounded-sm px-2 py-1 leading-relaxed whitespace-pre-wrap">
            <span className="text-foreground-subtle block">{t("Note for the reviewer")}</span>
            {review.reviewerNote}
          </p>
        )}
      </div>
      <span className="text-2xs text-muted-foreground shrink-0">
        {formatDistanceToNow(new Date(review.createdAt * 1000), { addSuffix: true })}
      </span>
    </li>
  );
}

function RoundHeader({ round }: { round: ReviewRound }) {
  const t = useT();
  return (
    <div className="bg-muted/40 text-2xs flex items-center justify-between gap-2 border-b px-3 py-1">
      <span className="font-medium">{t("Round {0}", round.round)}</span>
      <span className="text-muted-foreground">
        {round.baseVersionNumber > 0
          ? t("against approved v{0}", round.baseVersionNumber)
          : t("first approval")}
        {round.outcome === null && ` ${t("· open")}`}
      </span>
    </div>
  );
}

/**
 * A template's or a library's review conversation, newest round first, each round read top to
 * bottom.
 */
export function ReviewHistoryView({
  reviews,
  isLoading,
  isError,
  emptyText,
}: {
  reviews: readonly ReviewEntryData[] | undefined;
  isLoading: boolean;
  isError: boolean;
  /** Said when nothing was ever submitted. */
  emptyText: string;
}) {
  const t = useT();
  const rounds = useMemo(() => (reviews ? groupReviewRounds(reviews) : []), [reviews]);

  if (isLoading) {
    return (
      <div className="space-y-1.5 rounded-md border p-3">
        <Skeleton className="h-3.5 w-32" />
        <Skeleton className="h-10 w-full" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="text-muted-foreground rounded-md border px-3 py-2 text-xs">
        {t("The review history could not be loaded.")}
      </div>
    );
  }

  if (rounds.length === 0) {
    return (
      <div className="text-muted-foreground flex items-center gap-1.5 rounded-md border px-3 py-2 text-xs">
        <ClockRewindIcon className="size-3.5" />
        {emptyText}
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-md border">
      <div className="flex items-center gap-1.5 border-b px-3 py-2 text-xs font-semibold">
        <ClockRewindIcon className="size-3.5" />
        {t("Review history")}
      </div>
      {rounds.map((round) => (
        <div key={round.round}>
          <RoundHeader round={round} />
          <ul className="divide-y">
            {round.entries.map((review) => (
              <ReviewEntry key={review.id} review={review} />
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}

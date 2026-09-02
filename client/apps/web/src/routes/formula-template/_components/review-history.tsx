import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaTemplateReview } from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { formatDistanceToNow } from "date-fns";
import { HistoryIcon } from "lucide-react";
import { useMemo } from "react";
import { describeReviewDecision, groupReviewRounds, type ReviewRound } from "./review-rounds";

const TONE_CLASSES: Record<ReturnType<typeof describeReviewDecision>["tone"], string> = {
  neutral: "bg-sky-500/15 text-sky-700 dark:text-sky-300",
  positive: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  negative: "bg-destructive/15 text-destructive",
  warning: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  muted: "bg-muted text-muted-foreground",
};

function actorName(review: FormulaTemplateReview): string {
  if (review.actor?.name) return review.actor.name;
  if (review.decision === "Expired") return "System";
  return "Someone";
}

function ReviewEntry({ review }: { review: FormulaTemplateReview }) {
  const decision = describeReviewDecision(review.decision);
  return (
    <li className="flex items-start justify-between gap-3 px-3 py-1.5 text-xs">
      <div className="min-w-0 space-y-0.5">
        <div className="flex items-center gap-1.5">
          <Badge
            variant="outline"
            className={cn("text-2xs border-transparent px-1 py-0", TONE_CLASSES[decision.tone])}
          >
            {decision.label}
          </Badge>
          <span className="font-medium">{actorName(review)}</span>
        </div>
        {review.comment && <p className="text-muted-foreground">{review.comment}</p>}
      </div>
      <span className="text-2xs text-muted-foreground shrink-0">
        {formatDistanceToNow(new Date(review.createdAt * 1000), { addSuffix: true })}
      </span>
    </li>
  );
}

function RoundHeader({ round }: { round: ReviewRound }) {
  return (
    <div className="bg-muted/40 text-2xs flex items-center justify-between gap-2 border-b px-3 py-1">
      <span className="font-medium">Round {round.round}</span>
      <span className="text-muted-foreground">
        {round.baseVersionNumber > 0
          ? `against approved v${round.baseVersionNumber}`
          : "first approval"}
        {round.outcome === null && " · open"}
      </span>
    </div>
  );
}

/** The template's review conversation, newest round first. */
export function ReviewHistory({ templateId }: { templateId: string }) {
  const { data, isLoading, isError } = useQuery({
    ...queries.formulaTemplate.reviews(templateId),
    enabled: !!templateId,
    staleTime: 0,
  });

  const rounds = useMemo(() => (data ? groupReviewRounds(data) : []), [data]);

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
        The review history could not be loaded.
      </div>
    );
  }

  if (rounds.length === 0) {
    return (
      <div className="text-muted-foreground flex items-center gap-1.5 rounded-md border px-3 py-2 text-xs">
        <HistoryIcon className="size-3.5" />
        This template has never been submitted for review.
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-md border">
      <div className="flex items-center gap-1.5 border-b px-3 py-2 text-xs font-semibold">
        <HistoryIcon className="size-3.5" />
        Review history
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

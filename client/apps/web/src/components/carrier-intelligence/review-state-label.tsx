import { INTEL_EMPTY_VALUE } from "@/lib/carrier-intelligence";
import type { CarrierIntelReviewState } from "@trenova/graphql/generated/graphql";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { StatusDot, type StatusTone } from "./status-dot";

export type ReviewStateLabelProps = {
  state: CarrierIntelReviewState;
  reviewedAt?: number | null;
  showDot?: boolean;
  className?: string;
};

const REVIEW_TONE: Record<CarrierIntelReviewState, StatusTone> = {
  None: "neutral",
  NeedsReview: "medium",
  Reviewed: "success",
};

export function ReviewStateLabel({
  state,
  reviewedAt,
  showDot = false,
  className,
}: ReviewStateLabelProps) {
  const t = useT();
  const text: Record<CarrierIntelReviewState, string> = {
    None: t("No review needed"),
    NeedsReview: t("Needs review"),
    Reviewed: t("Reviewed"),
  };
  const description: Record<CarrierIntelReviewState, string> = {
    None: t("Nothing in the latest vetting asks for a manual review."),
    NeedsReview: t("A change or finding needs someone to look at it and sign off."),
    Reviewed: reviewedAt
      ? t("Reviewed {0}.", formatUnixDateTimeMedium(reviewedAt))
      : t("Someone reviewed the latest vetting."),
  };

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            className={cn(
              "inline-flex cursor-default items-center gap-2 text-xs",
              state === "NeedsReview" ? "text-foreground" : "text-muted-foreground",
              className,
            )}
            data-review-state={state}
          />
        }
      >
        {showDot ? <StatusDot tone={REVIEW_TONE[state]} /> : null}
        {text[state]}
      </TooltipTrigger>
      <TooltipContent>{description[state]}</TooltipContent>
    </Tooltip>
  );
}

export function ReviewRequiredLabel({
  required,
  className,
}: {
  required: boolean;
  className?: string;
}) {
  const t = useT();
  if (!required) {
    return <span className={cn("text-muted-foreground", className)}>{INTEL_EMPTY_VALUE}</span>;
  }
  return (
    <span
      className={cn("inline-flex items-center gap-2 text-xs", className)}
      title={t("A change or finding needs someone to look at it and sign off.")}
    >
      <StatusDot tone="medium" />
      {t("Review required")}
    </span>
  );
}

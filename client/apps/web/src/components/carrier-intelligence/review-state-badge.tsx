import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelReviewState } from "@trenova/graphql/generated/graphql";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { BadgeAttrProps } from "@trenova/shared/components/status-badge";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

export type ReviewStateBadgeProps = {
  state: CarrierIntelReviewState;
  reviewedAt?: number | null;
  className?: string;
};

export function ReviewStateBadge({ state, reviewedAt, className }: ReviewStateBadgeProps) {
  const t = useT();

  const attributes: Record<CarrierIntelReviewState, BadgeAttrProps> = {
    None: {
      variant: "secondary",
      text: t("No review needed"),
      description: t("Nothing in the latest vetting asks for a manual review."),
    },
    NeedsReview: {
      variant: "warning",
      text: t("Needs review"),
      description: t("A change or finding needs someone to look at it and sign off."),
    },
    Reviewed: {
      variant: "active",
      text: t("Reviewed"),
      description: reviewedAt
        ? t("Reviewed {0}.", formatUnixDateTimeMedium(reviewedAt))
        : t("Someone reviewed the latest vetting."),
    },
  };

  const attrs = attributes[state];

  return (
    <Badge variant={attrs.variant} className={cn("max-h-5", className)} title={attrs.description}>
      {attrs.text}
    </Badge>
  );
}

export type ReviewRequiredBadgeProps = {
  required: boolean;
  className?: string;
};

export function ReviewRequiredBadge({ required, className }: ReviewRequiredBadgeProps) {
  const t = useT();

  if (!required) {
    return <span className={cn("text-muted-foreground", className)}>-</span>;
  }

  return (
    <Badge
      variant="warning"
      className={cn("max-h-5", className)}
      title={t("A change or finding needs someone to look at it and sign off.")}
    >
      {t("Review required")}
    </Badge>
  );
}

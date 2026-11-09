import { HoverGlideGroup } from "@/components/hover-glide";
import type { AuditReview } from "@/lib/graphql/audit-overview";
import { audit } from "@/lib/queries/audit";
import { useQuery } from "@tanstack/react-query";
import { CheckCircleIcon, ShieldTickIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { REVIEW_KIND_ICONS, REVIEW_KIND_LABELS, reviewTone } from "./audit-review-kinds";
import { useReviewDecision } from "./use-review-decision";

const REVIEWED_PREVIEW = 1;

const WELL_TONE = {
  danger: "border-danger-border bg-danger-subtle text-danger-foreground",
  warning: "border-warning-border bg-warning-subtle text-warning-foreground",
  success: "border-success-border bg-success-subtle text-success-foreground",
} as const;

export function ReviewWell({
  tone,
  children,
}: {
  tone: keyof typeof WELL_TONE;
  children: React.ReactNode;
}) {
  return (
    <span
      aria-hidden
      className={cn("grid size-7 shrink-0 place-items-center rounded-md border", WELL_TONE[tone])}
    >
      {children}
    </span>
  );
}

function ReviewItem({ review, onOpen }: { review: AuditReview; onOpen: () => void }) {
  const t = useT();
  const now = useNow("minute");
  const decision = useReviewDecision(review);
  const Icon = REVIEW_KIND_ICONS[review.kind];
  const meta = [
    formatRelativeTime(review.occurredAt - now),
    review.offHours ? t("off hours") : null,
    REVIEW_KIND_LABELS[review.kind],
  ].filter(Boolean);

  return (
    <li
      data-glide-target
      className="relative grid grid-cols-[1.75rem_minmax(0,1fr)_auto] items-center gap-2.5 rounded-md px-2 py-1.5"
    >
      <ReviewWell tone={reviewTone(review.kind)}>
        <Icon className="size-3.5" />
      </ReviewWell>
      <button type="button" onClick={onOpen} className="ui-focus-ring min-w-0 rounded-sm text-left">
        <span className="block truncate text-sm font-medium">{review.title}</span>
        <span className="text-muted-foreground block truncate text-xs">{meta.join(" · ")}</span>
      </button>
      <Button
        type="button"
        size="xs"
        variant="outline"
        isLoading={decision.marking}
        onClick={() => void decision.markReviewed()}
      >
        {t("Mark reviewed")}
      </Button>
    </li>
  );
}

/**
 * The alert rules' open hits, latest first, each opening the entry that set it off. The card keeps
 * its place when nothing is open, saying what the rules watch.
 */
export function AuditReviewCard({
  reviews,
  openCount,
  onOpenReview,
}: {
  reviews: readonly AuditReview[];
  openCount: number;
  onOpenReview: (review: AuditReview) => void;
}) {
  const t = useT();
  const reviewed = useQuery(audit.reviews(REVIEWED_PREVIEW));
  const latest = reviewed.data?.reviewed[0];

  return (
    <section
      aria-label={t("Needs a review")}
      className="bg-card flex min-w-0 flex-col rounded-md border"
    >
      <header className="flex items-start gap-2 px-3 pt-2.5">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold">{t("Needs a review")}</h3>
          <p className="text-muted-foreground text-xs">
            {t("Raised by your alert rules. Open one to see the entry.")}
          </p>
        </div>
        <Badge variant={openCount > 0 ? "warning" : "success"} className="ml-auto shrink-0">
          {t("{0, plural, one {# open} other {# open}}", openCount)}
        </Badge>
      </header>
      {reviews.length > 0 ? (
        <HoverGlideGroup className="bg-card px-1 py-1">
          <ul className="flex flex-col gap-px">
            {reviews.map((review) => (
              <ReviewItem key={review.id} review={review} onOpen={() => onOpenReview(review)} />
            ))}
          </ul>
        </HoverGlideGroup>
      ) : (
        <div className="flex items-start gap-2.5 px-3 py-3">
          <ReviewWell tone="success">
            <ShieldTickIcon className="size-3.5" />
          </ReviewWell>
          <div className="min-w-0">
            <p className="text-sm font-medium">{t("Nothing needs a review")}</p>
            <p className="text-muted-foreground text-xs">
              {t(
                "Your alert rules watch role and API key changes, two-factor resets, sign-in settings, bulk deletes and bulk exports.",
              )}
            </p>
          </div>
        </div>
      )}
      {openCount > reviews.length ? (
        <p className="text-muted-foreground px-3 pb-1 text-xs">
          {t("{0, plural, one {# more open} other {# more open}}", openCount - reviews.length)}
        </p>
      ) : null}
      <footer className="text-muted-foreground mx-3 mt-auto flex items-center gap-1.5 border-t py-2 text-xs">
        <CheckCircleIcon aria-hidden className="size-3" />
        {reviewed.data && latest ? (
          <span className="truncate">
            {t(
              "{0, plural, one {# reviewed this week} other {# reviewed this week}}, latest by {1}",
              reviewed.data.reviewedCount,
              latest.reviewedBy?.name ?? t("someone"),
            )}
          </span>
        ) : (
          <span>{t("None reviewed this week")}</span>
        )}
      </footer>
    </section>
  );
}

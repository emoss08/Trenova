import type { PerformanceReviewRow } from "@/lib/graphql/performance-review";

export type ReviewCadence = {
  /** When the next review falls due, from the last closed review; null before the first. */
  nextDueAt: number | null;
  lastReviewedAt: number | null;
  templateName: string | null;
  cadenceMonths: number | null;
};

/** The closed reviews, newest first. */
export function closedReviews(reviews: readonly PerformanceReviewRow[]): PerformanceReviewRow[] {
  return reviews
    .filter((review) => review.status === "Closed")
    .sort((a, b) => (b.closedAt ?? b.periodEnd) - (a.closedAt ?? a.periodEnd));
}

/**
 * The cadence the reviews keep: the server schedules the next review when one closes, so the
 * newest closed review carries the next due date and the template its rhythm. How far off that
 * date is belongs to the leaf that prints it, which keeps the clock.
 */
export function reviewCadence(
  reviews: readonly PerformanceReviewRow[],
  closed: readonly PerformanceReviewRow[] = closedReviews(reviews),
): ReviewCadence {
  const last = closed[0] ?? null;
  const template = last?.template ?? reviews[0]?.template ?? null;
  return {
    nextDueAt: last?.nextReviewAt ?? null,
    lastReviewedAt: last ? (last.acknowledgedAt ?? last.closedAt ?? null) : null,
    templateName: template?.name ?? null,
    cadenceMonths: template?.cadenceMonths ?? null,
  };
}

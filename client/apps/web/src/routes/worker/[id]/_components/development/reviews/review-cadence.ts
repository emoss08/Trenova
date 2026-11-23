import type { PerformanceReviewRow } from "@/lib/graphql/performance-review";

const DAY = 86_400;

export type ReviewCadence = {
  /** When the next review falls due, from the last closed review; null before the first. */
  nextDueAt: number | null;
  /** Whole days from now to that date; negative once it has passed. */
  daysUntilDue: number | null;
  lastReviewedAt: number | null;
  templateName: string | null;
  cadenceMonths: number | null;
};

/**
 * The cadence the reviews keep: the server schedules the next review when one closes, so the
 * newest closed review carries the next due date and the template its rhythm.
 */
export function reviewCadence(
  reviews: readonly PerformanceReviewRow[],
  now: number,
): ReviewCadence {
  const closed = reviews
    .filter((review) => review.status === "Closed")
    .sort((a, b) => (b.closedAt ?? b.periodEnd) - (a.closedAt ?? a.periodEnd));
  const last = closed[0] ?? null;
  const template = last?.template ?? reviews[0]?.template ?? null;
  const nextDueAt = last?.nextReviewAt ?? null;
  return {
    nextDueAt,
    daysUntilDue: nextDueAt === null ? null : Math.floor((nextDueAt - now) / DAY),
    lastReviewedAt: last ? (last.acknowledgedAt ?? last.closedAt ?? null) : null,
    templateName: template?.name ?? null,
    cadenceMonths: template?.cadenceMonths ?? null,
  };
}

import type { FormulaReviewDecision } from "@trenova/shared/types/formula-template";

/** One entry of a template's or a library's review conversation. */
export type ReviewEntryData = {
  id: string;
  round: number;
  decision: FormulaReviewDecision;
  comment: string;
  reviewerNote: string;
  baseVersionNumber: number;
  createdAt: number;
  actor?: { name: string } | null;
  requestedReviewer?: { name: string } | null;
};

export type ReviewDecisionPresentation = {
  label: string;
  tone: "neutral" | "positive" | "negative" | "warning" | "muted";
};

const DECISIONS: Record<FormulaReviewDecision, ReviewDecisionPresentation> = {
  Submitted: { label: "Submitted", tone: "neutral" },
  Approved: { label: "Approved", tone: "positive" },
  Rejected: { label: "Rejected", tone: "negative" },
  ChangesRequested: { label: "Changes requested", tone: "warning" },
  Expired: { label: "Expired", tone: "muted" },
};

export function describeReviewDecision(
  decision: FormulaReviewDecision,
): ReviewDecisionPresentation {
  return DECISIONS[decision];
}

export type ReviewRound = {
  round: number;
  entries: ReviewEntryData[];
  /** The approved version the round's diff was taken against; 0 means first approval. */
  baseVersionNumber: number;
  /** The last decision in the round, or null while it awaits one. */
  outcome: FormulaReviewDecision | null;
};

/**
 * Groups history entries (newest first, as the server returns them) into
 * rounds, newest round first, with entries inside each round oldest first so
 * the conversation reads top to bottom.
 */
export function groupReviewRounds(reviews: readonly ReviewEntryData[]): ReviewRound[] {
  const byRound = new Map<number, ReviewEntryData[]>();
  for (const review of reviews) {
    const entries = byRound.get(review.round) ?? [];
    entries.push(review);
    byRound.set(review.round, entries);
  }

  return [...byRound.entries()]
    .sort(([a], [b]) => b - a)
    .map(([round, entries]) => {
      const ordered = [...entries].sort((a, b) => a.createdAt - b.createdAt);
      const last = ordered[ordered.length - 1];
      const closed =
        last.decision === "Approved" || last.decision === "Rejected" || last.decision === "Expired";
      return {
        round,
        entries: ordered,
        baseVersionNumber: ordered[0]?.baseVersionNumber ?? 0,
        outcome: closed ? last.decision : null,
      };
    });
}

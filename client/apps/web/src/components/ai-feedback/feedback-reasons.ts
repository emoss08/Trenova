import type { AiFeedbackRating, AiFeedbackReason } from "@/lib/graphql/ai-feedback";

export type FeedbackReasonOption = {
  value: AiFeedbackReason;
  /** English source string; translated where it is drawn. */
  label: string;
};

/** The most a comment may hold, matching the server's limit. */
export const FEEDBACK_COMMENT_LIMIT = 1000;

export const NEGATIVE_REASONS: readonly FeedbackReasonOption[] = [
  { value: "Inaccurate", label: "Inaccurate" },
  { value: "MadeUpNumbers", label: "Made-up numbers" },
  { value: "Incomplete", label: "Incomplete" },
  { value: "WrongAction", label: "Wrong action" },
  { value: "IgnoredInstructions", label: "Ignored instructions" },
  { value: "NotRelevant", label: "Not relevant" },
  { value: "HardToRead", label: "Hard to read" },
  { value: "Unsafe", label: "Unsafe" },
  { value: "Other", label: "Other" },
];

export const POSITIVE_REASONS: readonly FeedbackReasonOption[] = [
  { value: "Accurate", label: "Accurate" },
  { value: "Helpful", label: "Helpful" },
  { value: "SavedTime", label: "Saved time" },
];

/** The reasons that may accompany a rating: each belongs to one side. */
export function reasonsFor(rating: AiFeedbackRating): readonly FeedbackReasonOption[] {
  return rating === 1 ? POSITIVE_REASONS : NEGATIVE_REASONS;
}

/** Keeps only the reasons on the rating's side, in the order they are offered. */
export function reasonsMatching(
  rating: AiFeedbackRating,
  reasons: readonly AiFeedbackReason[],
): AiFeedbackReason[] {
  const chosen = new Set(reasons);

  return reasonsFor(rating)
    .filter((option) => chosen.has(option.value))
    .map((option) => option.value);
}

/** Adds a reason that is not chosen and removes one that is. */
export function toggleReason(
  reasons: readonly AiFeedbackReason[],
  reason: AiFeedbackReason,
): AiFeedbackReason[] {
  return reasons.includes(reason)
    ? reasons.filter((value) => value !== reason)
    : [...reasons, reason];
}

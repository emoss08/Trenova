export type ExplanationStatus = "none" | "fresh" | "stale";

/**
 * Whether an AI explanation still describes the expression on screen. The
 * comparison ignores surrounding whitespace: reformatting is not a change in
 * meaning, and telling someone their explanation is stale because they added
 * a newline would teach them to ignore the notice.
 */
export function explanationStatus({
  expression,
  explainedFor,
  hasExplanation,
}: {
  expression: string;
  explainedFor: string | null;
  hasExplanation: boolean;
}): ExplanationStatus {
  if (!hasExplanation || explainedFor === null) return "none";
  return expression.trim() === explainedFor.trim() ? "fresh" : "stale";
}

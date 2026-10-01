import type { DecisionRowView } from "./decision-presenters";

/**
 * The rows a batch could take along with the focused one: every proposal of
 * the same tool that is decided as proposed, the focused row included, in the
 * queue's order. Fewer than two is no batch, and a plan is never one.
 */
export function rowsLike(rows: readonly DecisionRowView[], focusedId: string | null): string[] {
  const focused = rows.find((row) => row.id === focusedId);
  if (!focused || !focused.batchable) {
    return [];
  }

  const like = rows
    .filter((row) => row.batchable && row.toolName === focused.toolName)
    .map((row) => row.id);

  return like.length >= 2 ? like : [];
}

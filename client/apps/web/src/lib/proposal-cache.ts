import type { QueryClient } from "@tanstack/react-query";

/**
 * The queries a decision on a proposal or a plan makes stale, whichever
 * surface it was made on.
 *
 * The chat and AI Control read the same records through different queries.
 * Each used to refresh only its own, so a card approved in the chat stayed
 * clickable in AI Control until a remount, and the other way round.
 */
export async function invalidateProposalViews(queryClient: QueryClient, threadId?: string) {
  const keys: unknown[][] = [
    ["assistant", "pending-proposals"],
    ["agent-proposal-list"],
    ["agent-plan-list"],
    ["agent-run-list"],
    ["pending-decisions"],
    ["pending-decision-summary"],
    ["attention"],
  ];
  if (threadId) {
    keys.push(
      ["assistant-proposals", threadId],
      ["assistant-plans", threadId],
      ["assistant-messages", threadId],
      ["assistant-artifacts", threadId],
    );
  } else {
    keys.push(["assistant-proposals"], ["assistant-plans"], ["assistant-artifacts"]);
  }

  await Promise.all(keys.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}

import type { AssistantProposal, ProposalDecision } from "@/types/assistant";
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

/**
 * The status the server records for a decision.
 *
 * Deciding and executing are two steps: the resolve call records the decision
 * and returns, and the tool runs after it. So an approval's immediate truth is
 * "accepted, not yet run", which is exactly what the card should show while it
 * waits.
 */
export function decidedStatus(decision: ProposalDecision): AssistantProposal["status"] {
  switch (decision) {
    case "Accepted":
      return "Accepted";
    case "Modified":
      return "Modified";
    default:
      return "Rejected";
  }
}

/** One proposal's decision written into a list, leaving the rest alone. */
export function applyDecision(
  proposals: readonly AssistantProposal[],
  proposalId: string,
  decision: ProposalDecision,
): AssistantProposal[] {
  return proposals.map((proposal) =>
    proposal.id === proposalId ? { ...proposal, status: decidedStatus(decision) } : proposal,
  );
}

/**
 * Writes a decision into every cached list that holds the proposal, so the
 * card stops being a question the moment the server accepts the decision.
 *
 * Without this the card kept its buttons until a refetch came back, and a
 * second click in that window reached a proposal that had already been
 * decided — which the server refuses, correctly, with an error that reads
 * like a bug: "this proposal has already been decided: it is executed". The
 * invalidation still runs; this only closes the window it leaves open.
 */
export function markProposalDecided(
  queryClient: QueryClient,
  proposalId: string,
  decision: ProposalDecision,
) {
  queryClient.setQueriesData<{ results: AssistantProposal[] }>(
    { queryKey: ["assistant-proposals"] },
    (data) =>
      data === undefined
        ? data
        : { ...data, results: applyDecision(data.results, proposalId, decision) },
  );
}

import { queries } from "@/lib/queries";
import type {
  AssistantPlan,
  AssistantProposal,
  PlanDecision,
  ProposalDecision,
} from "@/types/assistant";
import type { QueryClient } from "@tanstack/react-query";

/*
Every key here comes from the query factory rather than being written out.

createQueryKeys prepends its scope and the method name, so the thread
proposals a component reads under `queries.assistant.proposals(id)` actually
live at ["assistant", "proposals", "assistant-proposals", id]. This file used
to invalidate ["assistant-proposals", id], which is a prefix of nothing:
TanStack matches from the start of the key, and the start is "assistant".

Nothing failed. The request succeeded, the mutation resolved, and the cache
was never touched — so an approved proposal kept its buttons until the thread
was remounted, which is indistinguishable from the click not working.
*/

/** The prefix every thread's entry for one factory key shares. */
function scopeOf(key: { _def: readonly unknown[] }): unknown[] {
  return [...key._def];
}

/**
 * The queries a decision on a proposal or a plan makes stale, whichever
 * surface it was made on.
 *
 * The chat and AI Control read the same records through different queries.
 * Each used to refresh only its own, so a card approved in the chat stayed
 * clickable in AI Control until a remount, and the other way round.
 */
export async function invalidateProposalViews(queryClient: QueryClient, threadId?: string) {
  // These are read with a plain useQuery, so their key is what it says.
  const keys: unknown[][] = [
    ["agent-proposal-list"],
    ["agent-plan-list"],
    ["agent-run-list"],
    ["pending-decisions"],
    ["pending-decision-summary"],
    ["attention"],
    // A decision can change the person's own home page.
    [...queries.homeLayout.effective().queryKey],
    // What a write would do is read from the world as it is now, and a
    // decision changes it: the one decided now reads as recorded, and a later
    // step or a proposal on the same record starts from what this one left.
    scopeOf(queries.agentPreview),
  ];

  const threadScoped = [
    queries.assistant.proposals,
    queries.assistant.plans,
    queries.assistant.messages,
    queries.assistant.artifacts,
  ];
  for (const key of threadScoped) {
    keys.push(threadId ? [...key(threadId).queryKey] : scopeOf(key));
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
    { queryKey: scopeOf(queries.assistant.proposals) },
    (data) =>
      data === undefined
        ? data
        : { ...data, results: applyDecision(data.results, proposalId, decision) },
  );
}

/**
 * Writes a plan's decision into every cached list of plans, so the approval
 * box moves on the moment the server accepts it rather than when a refetch
 * comes back. The invalidation still runs and brings the steps' outcomes.
 */
export function markPlanDecided(queryClient: QueryClient, planId: string, decision: PlanDecision) {
  queryClient.setQueriesData<{ results: AssistantPlan[] }>(
    { queryKey: scopeOf(queries.assistant.plans) },
    (data) =>
      data === undefined
        ? data
        : {
            ...data,
            results: data.results.map((plan) =>
              plan.id === planId
                ? { ...plan, status: decision === "Accepted" ? "Approved" : "Rejected" }
                : plan,
            ),
          },
  );
}

/**
 * Writes a status into every cached list that holds these proposals: an
 * approval moving into its undo window, or back out of it on undo, so the
 * approval box follows the click rather than the refetch.
 */
export function markProposalsStatus(
  queryClient: QueryClient,
  proposalIds: readonly string[],
  status: AssistantProposal["status"],
) {
  const ids = new Set(proposalIds);
  queryClient.setQueriesData<{ results: AssistantProposal[] }>(
    { queryKey: scopeOf(queries.assistant.proposals) },
    (data) =>
      data === undefined
        ? data
        : {
            ...data,
            results: data.results.map((proposal) =>
              ids.has(proposal.id) ? { ...proposal, status } : proposal,
            ),
          },
  );
}

/** Writes a plan's status into every cached list of plans. */
export function markPlanStatus(
  queryClient: QueryClient,
  planId: string,
  status: AssistantPlan["status"],
) {
  queryClient.setQueriesData<{ results: AssistantPlan[] }>(
    { queryKey: scopeOf(queries.assistant.plans) },
    (data) =>
      data === undefined
        ? data
        : {
            ...data,
            results: data.results.map((plan) => (plan.id === planId ? { ...plan, status } : plan)),
          },
  );
}

/**
 * Writes the edits a person saved on a pending proposal into every cached
 * list that holds it, so a draft shows the saved wording as soon as the
 * server keeps it rather than when a refetch comes back. Null clears them.
 */
export function markProposalEdits(
  queryClient: QueryClient,
  proposalId: string,
  pendingModifications: Record<string, unknown> | null,
) {
  queryClient.setQueriesData<{ results: AssistantProposal[] }>(
    { queryKey: scopeOf(queries.assistant.proposals) },
    (data) =>
      data === undefined
        ? data
        : {
            ...data,
            results: data.results.map((proposal) =>
              proposal.id === proposalId ? { ...proposal, pendingModifications } : proposal,
            ),
          },
  );
}

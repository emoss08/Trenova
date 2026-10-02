import { decideMyProposals } from "@/lib/graphql/agent-decisions";
import { invalidateProposalViews, markProposalDecided } from "@/lib/proposal-cache";
import type { AssistantProposal } from "@/types/assistant";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type { ApprovalEntry } from "./approval-queue";
import { useDecisionFollowUp } from "./decision-follow-up";
import { batchPreviewDigests } from "./proposal-preview/preview-gate";
import { presentProposal } from "./proposal-presenters";

/** Why a batch was turned down in a conversation, as the server records it. */
const BATCH_REJECT_REASON = "rejected_in_conversation";

/** How a batch went: how many of its changes the server took, and why the rest did not go. */
export type BatchOutcome = { approved: number; total: number; errors: string[] };

/**
 * What every kind of decision does once the server has recorded it: the
 * views of the conversation's proposals are read again, the conversation is
 * told to pick up the turn in which the agent answers, and the caller hears
 * which entry was settled.
 */
export function useAfterDecision(
  threadId: string,
  entry: ApprovalEntry,
  onDecided?: (entry: ApprovalEntry) => void,
) {
  const queryClient = useQueryClient();
  const followUp = useDecisionFollowUp();

  return useCallback(
    async (anchorId: string) => {
      await invalidateProposalViews(queryClient, threadId);
      // The server starts the turn in which the agent answers; the
      // conversation picks it up rather than sending anything itself, so a
      // note to the agent is answered once, by that turn.
      followUp?.(anchorId);
      onDecided?.(entry);
    },
    [entry, followUp, onDecided, queryClient, threadId],
  );
}

/**
 * Decides several changes of one kind together. Approving sends the digest
 * of each preview the person was shown, so a change approved without being
 * opened is recorded as approved unreviewed, which is the truth. Every change
 * the server took is marked decided in the cache straight away.
 */
export async function decideBatch(
  queryClient: QueryClient,
  proposals: readonly AssistantProposal[],
  {
    approving,
    shownDigests,
    note,
  }: { approving: boolean; shownDigests: ReadonlyMap<string, string>; note?: string },
): Promise<BatchOutcome> {
  const ids = proposals.map((proposal) => proposal.id);
  const results = await decideMyProposals(ids, {
    decision: approving ? "Accepted" : "Rejected",
    reasonCode: approving ? undefined : BATCH_REJECT_REASON,
    previewDigests: approving ? batchPreviewDigests(ids, shownDigests) : undefined,
    note: approving ? undefined : note,
  });
  const failed = results.filter((result) => (result.error ?? "") !== "");
  for (const result of results) {
    if ((result.error ?? "") === "" || result.decision) {
      markProposalDecided(queryClient, result.proposalId, approving ? "Accepted" : "Rejected");
    }
  }

  return {
    approved: results.length - failed.length,
    total: results.length,
    errors: failed.map((result) => {
      const proposal = proposals.find((candidate) => candidate.id === result.proposalId);
      const summary = proposal ? presentProposal(proposal).summary : "";
      return summary === "" ? (result.error ?? "") : `${summary}: ${result.error ?? ""}`;
    }),
  };
}

import type { AssistantProposal } from "@/types/assistant";
import { classifyProposal } from "./proposal-state";

/** Waiting proposals of one tool that can be approved together. */
export type ProposalBatch = {
  toolName: string;
  proposals: AssistantProposal[];
};

/** The most proposals one batch decision takes; the server refuses more. */
export const MAX_BATCH_PROPOSALS = 50;

/**
 * Groups the proposals that stand on their own and still wait on the person
 * by tool, keeping only the groups of two or more. A plan's steps are decided
 * with their plan and never land here. The order is the conversation's.
 */
export function batchableProposals(proposals: readonly AssistantProposal[]): ProposalBatch[] {
  const byTool = new Map<string, AssistantProposal[]>();
  for (const proposal of proposals) {
    if (proposal.planId !== "" || classifyProposal(proposal) !== "awaiting") {
      continue;
    }
    const group = byTool.get(proposal.toolName);
    if (group) {
      group.push(proposal);
    } else {
      byTool.set(proposal.toolName, [proposal]);
    }
  }

  const batches: ProposalBatch[] = [];
  for (const [toolName, group] of byTool) {
    if (group.length >= 2) {
      batches.push({ toolName, proposals: group.slice(0, MAX_BATCH_PROPOSALS) });
    }
  }

  return batches;
}

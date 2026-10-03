import {
  CommitMyDecisionNowDocument,
  DecideAgentPlanDocument,
  DecideAgentProposalDocument,
  DecideMyPlanDocument,
  DecideMyProposalDocument,
  DecideMyProposalsDocument,
  ResolveAgentExceptionDocument,
  UndoMyDecisionDocument,
  type AgentExceptionResolveInput,
  type AgentPlanDecisionInput,
  type AgentProposalDecisionInput,
  type DecideAgentProposalsInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

/**
 * Decides any proposal, as an approver. Needs permission to update agent
 * proposals, so only AI Control and the decisions queue call it.
 */
export async function decideAgentProposal(id: string, input: AgentProposalDecisionInput) {
  const data = await requestGraphQL({
    document: DecideAgentProposalDocument,
    operationName: "DecideAgentProposal",
    variables: { id, input },
  });

  return data.decideAgentProposal;
}

/**
 * Decides a proposal raised in one of the person's own conversations. It
 * needs only the assistant, so anyone who can ask an agent can answer what
 * it asks them; a proposal from someone else's conversation is not found.
 */
export async function decideMyProposal(id: string, input: AgentProposalDecisionInput) {
  const data = await requestGraphQL({
    document: DecideMyProposalDocument,
    operationName: "DecideMyProposal",
    variables: { id, input },
  });

  return data.decideMyProposal;
}

/**
 * Decides several proposals of one tool raised in the person's own
 * conversations, each as proposed and each against the digest of the preview
 * the person was shown. The batch is refused whole when one is not theirs;
 * once it runs, each proposal reports what became of it.
 */
export async function decideMyProposals(ids: string[], input: DecideAgentProposalsInput) {
  const data = await requestGraphQL({
    document: DecideMyProposalsDocument,
    operationName: "DecideMyProposals",
    variables: { ids, input },
  });

  return data.decideMyProposals;
}

/**
 * Decides a plan raised in one of the person's own conversations, by an agent
 * they may still use. Like `decideMyProposal` it needs only the assistant;
 * every step's write still runs only if the person may make it, and a plan
 * from someone else's conversation is not found.
 */
export async function decideMyPlan(id: string, input: AgentPlanDecisionInput) {
  const data = await requestGraphQL({
    document: DecideMyPlanDocument,
    operationName: "DecideMyPlan",
    variables: { id, input },
  });

  return data.decideMyPlan;
}

/**
 * Decides any plan, as an approver. Needs permission to update agent
 * proposals, so only AI Control and the decisions queue call it.
 */
export async function decideAgentPlan(id: string, input: AgentPlanDecisionInput) {
  const data = await requestGraphQL({
    document: DecideAgentPlanDocument,
    operationName: "DecideAgentPlan",
    variables: { id, input },
  });

  return data.decideAgentPlan;
}

export async function resolveAgentException(id: string, input: AgentExceptionResolveInput) {
  const data = await requestGraphQL({
    document: ResolveAgentExceptionDocument,
    operationName: "ResolveAgentException",
    variables: { id, input },
  });

  return data.resolveAgentException;
}

/** Which approval an undo or a "do it now" is about: a proposal (with its batch) or a plan. */
export type ApprovalTarget = { proposalId: string } | { planId: string };

function targetVariables(target: ApprovalTarget) {
  return "planId" in target
    ? { proposalId: null, planId: target.planId }
    : { proposalId: target.proposalId, planId: null };
}

/**
 * Takes back the person's approval while it is in its undo window; the change
 * waits on them again. Once it has gone through the server answers with a
 * conflict.
 */
export async function undoMyDecision(target: ApprovalTarget) {
  const data = await requestGraphQL({
    document: UndoMyDecisionDocument,
    operationName: "UndoMyDecision",
    variables: targetVariables(target),
  });

  return data.undoMyDecision;
}

/** Ends the undo window on the person's approval, so it goes through now. */
export async function commitMyDecisionNow(target: ApprovalTarget) {
  const data = await requestGraphQL({
    document: CommitMyDecisionNowDocument,
    operationName: "CommitMyDecisionNow",
    variables: targetVariables(target),
  });

  return data.commitMyDecisionNow;
}

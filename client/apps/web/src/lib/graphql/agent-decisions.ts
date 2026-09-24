import {
  DecideAgentPlanDocument,
  DecideAgentProposalDocument,
  DecideMyProposalDocument,
  ResolveAgentExceptionDocument,
  type AgentExceptionResolveInput,
  type AgentPlanDecisionInput,
  type AgentProposalDecisionInput,
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

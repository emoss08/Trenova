import {
  DecideAgentPlanDocument,
  DecideAgentProposalDocument,
  ResolveAgentExceptionDocument,
  type AgentExceptionResolveInput,
  type AgentPlanDecisionInput,
  type AgentProposalDecisionInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export async function decideAgentProposal(id: string, input: AgentProposalDecisionInput) {
  const data = await requestGraphQL({
    document: DecideAgentProposalDocument,
    operationName: "DecideAgentProposal",
    variables: { id, input },
  });

  return data.decideAgentProposal;
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

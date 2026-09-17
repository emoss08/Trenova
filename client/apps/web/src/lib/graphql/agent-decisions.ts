import {
  DecideAgentProposalDocument,
  ResolveAgentExceptionDocument,
  type AgentExceptionResolveInput,
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

export async function resolveAgentException(id: string, input: AgentExceptionResolveInput) {
  const data = await requestGraphQL({
    document: ResolveAgentExceptionDocument,
    operationName: "ResolveAgentException",
    variables: { id, input },
  });

  return data.resolveAgentException;
}

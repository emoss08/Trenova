import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentSafetyDocument,
  AgentSafetyFieldsFragmentDoc,
  AgentToolAutonomyFieldsFragmentDoc,
  AgentToolPoliciesDocument,
  AgentToolPolicyFieldsFragmentDoc,
  type AgentAutonomyAnswer,
  type AgentEgressClass,
  type AgentExternalRead,
  type AgentReachWarningKind,
  type AgentSafetyFieldsFragment,
  type AgentToolAutonomyFieldsFragment,
  type AgentToolPolicyFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentToolPolicy = AgentToolPolicyFieldsFragment;
export type AgentToolAutonomy = AgentToolAutonomyFieldsFragment;

export type AgentToolSafety = {
  policyName: string;
  clean: AgentToolAutonomy;
  tainted: AgentToolAutonomy;
};

export type AgentSafety = Omit<AgentSafetyFieldsFragment, "tools"> & {
  tools: AgentToolSafety[];
};

export type { AgentAutonomyAnswer, AgentEgressClass, AgentExternalRead, AgentReachWarningKind };

export async function fetchAgentToolPolicies(options?: {
  signal?: AbortSignal;
}): Promise<AgentToolPolicy[]> {
  const data = await requestGraphQL({
    document: AgentToolPoliciesDocument,
    operationName: "AgentToolPolicies",
    signal: options?.signal,
  });

  return data.agentToolPolicies.map((policy) =>
    getFragmentData(AgentToolPolicyFieldsFragmentDoc, policy),
  );
}

/** Every agent when agentIds is left out, otherwise only the ones named. */
export async function fetchAgentSafety(
  agentIds?: string[],
  options?: { signal?: AbortSignal },
): Promise<AgentSafety[]> {
  const data = await requestGraphQL({
    document: AgentSafetyDocument,
    operationName: "AgentSafety",
    variables: { agentIds: agentIds ?? null },
    signal: options?.signal,
  });

  return data.agentSafety.map((entry) => {
    const safety = getFragmentData(AgentSafetyFieldsFragmentDoc, entry);
    return {
      ...safety,
      tools: safety.tools.map((tool) => ({
        policyName: tool.policyName,
        clean: getFragmentData(AgentToolAutonomyFieldsFragmentDoc, tool.clean),
        tainted: getFragmentData(AgentToolAutonomyFieldsFragmentDoc, tool.tainted),
      })),
    };
  });
}

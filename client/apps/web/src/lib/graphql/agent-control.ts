import {
  AgentControlFieldsFragmentDoc,
  AgentControlSettingsDocument,
  UpdateAgentControlDocument,
  type AgentControlFieldsFragment,
  type AgentControlInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentControl = AgentControlFieldsFragment;

/** One key for the organization's switches, shared by every reader of them. */
export const AGENT_CONTROL_QUERY_KEY = ["agent-control"] as const;

export function agentControlQueryOptions() {
  return {
    queryKey: AGENT_CONTROL_QUERY_KEY,
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentControl({ signal }),
  };
}

export async function fetchAgentControl(options?: { signal?: AbortSignal }): Promise<AgentControl> {
  const data = await requestGraphQL({
    document: AgentControlSettingsDocument,
    operationName: "AgentControlSettings",
    signal: options?.signal,
  });

  return getFragmentData(AgentControlFieldsFragmentDoc, data.agentControl);
}

export async function updateAgentControl(input: AgentControlInput): Promise<AgentControl> {
  const data = await requestGraphQL({
    document: UpdateAgentControlDocument,
    operationName: "UpdateAgentControl",
    variables: { input },
  });

  return getFragmentData(AgentControlFieldsFragmentDoc, data.updateAgentControl);
}

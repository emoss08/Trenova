import {
  AgentControlFieldsFragmentDoc,
  AgentControlSettingsDocument,
  UpdateAgentControlDocument,
  type AgentControlFieldsFragment,
  type AgentControlInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/generated";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentControl = AgentControlFieldsFragment;

export async function fetchAgentControl(): Promise<AgentControl> {
  const data = await requestGraphQL({
    document: AgentControlSettingsDocument,
    operationName: "AgentControlSettings",
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

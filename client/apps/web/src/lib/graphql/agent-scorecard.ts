import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentScorecardDocument,
  AgentScorecardFieldsFragmentDoc,
  type AgentScorecardFieldsFragment,
  type AgentScorecardWindow,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentScorecard = AgentScorecardFieldsFragment;
export type AgentToolOutcome = AgentScorecard["byTool"][number];
export type AgentToolTrustRow = AgentScorecard["toolTrust"][number];
export type { AgentScorecardWindow };

export const SCORECARD_WINDOWS: AgentScorecardWindow[] = ["Last7Days", "Last30Days", "Last90Days"];

export async function fetchAgentScorecard(
  agentDefinitionId: string,
  window: AgentScorecardWindow,
  options?: { signal?: AbortSignal },
): Promise<AgentScorecard> {
  const data = await requestGraphQL({
    document: AgentScorecardDocument,
    operationName: "AgentScorecard",
    variables: { input: { agentDefinitionId, window } },
    signal: options?.signal,
  });

  return getFragmentData(AgentScorecardFieldsFragmentDoc, data.agentScorecard);
}

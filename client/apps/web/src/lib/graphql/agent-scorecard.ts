import { getFragmentData } from "@trenova/graphql/fragment-data";
import type {
  AgentScorecardFieldsFragment,
  AgentScorecardWindow,
} from "@trenova/graphql/generated/graphql";
import {
  AgentScorecardDocument,
  AgentScorecardFieldsFragmentDoc,
} from "@trenova/graphql/generated/documents/agent-scorecard/scorecard";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentScorecard = AgentScorecardFieldsFragment;
export type AgentToolOutcome = AgentScorecard["byTool"][number];
export type AgentToolTrustRow = AgentScorecard["toolTrust"][number];
export type AgentToolVerdictRow = AgentScorecard["toolVerdicts"][number];
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

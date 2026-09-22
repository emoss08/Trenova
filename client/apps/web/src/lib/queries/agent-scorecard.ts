import { fetchAgentScorecard, type AgentScorecardWindow } from "@/lib/graphql/agent-scorecard";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const agentScorecard = createQueryKeys("agentScorecard", {
  detail: (agentDefinitionId: string, window: AgentScorecardWindow) => ({
    queryKey: [agentDefinitionId, window],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAgentScorecard(agentDefinitionId, window, { signal }),
  }),
});

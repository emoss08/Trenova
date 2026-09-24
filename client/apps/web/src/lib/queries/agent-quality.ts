import {
  fetchAgentQuality,
  fetchAgentQualityAgents,
  fetchAgentQualityControl,
  fetchAgentQualityOverview,
  fetchAgentSuiteRunCases,
  fetchAgentSuiteRuns,
  fetchAgentWorstRatedAnswers,
  type QualityPageRequest,
} from "@/lib/graphql/agent-quality";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type Signal = { signal?: AbortSignal };

export const agentQuality = createQueryKeys("agentQuality", {
  overview: () => ({
    queryKey: ["overview"],
    queryFn: ({ signal }: Signal) => fetchAgentQualityOverview({ signal }),
  }),
  agents: (page: QualityPageRequest) => ({
    queryKey: [page.first, page.after ?? "", page.includeTotalCount],
    queryFn: ({ signal }: Signal) => fetchAgentQualityAgents(page, { signal }),
  }),
  worstRated: (agentDefinitionId: string | null, page: QualityPageRequest) => ({
    queryKey: [agentDefinitionId ?? "", page.first, page.after ?? ""],
    queryFn: ({ signal }: Signal) =>
      fetchAgentWorstRatedAnswers(agentDefinitionId, page, { signal }),
  }),
  agent: (agentDefinitionId: string) => ({
    queryKey: [agentDefinitionId],
    queryFn: ({ signal }: Signal) => fetchAgentQuality(agentDefinitionId, { signal }),
  }),
  suiteRuns: (agentDefinitionId: string, page: QualityPageRequest) => ({
    queryKey: [agentDefinitionId, page.first, page.after ?? "", page.includeTotalCount],
    queryFn: ({ signal }: Signal) => fetchAgentSuiteRuns(agentDefinitionId, page, { signal }),
  }),
  suiteRunCases: (suiteRunId: string, page: QualityPageRequest) => ({
    queryKey: [suiteRunId, page.first, page.after ?? "", page.includeTotalCount],
    queryFn: ({ signal }: Signal) => fetchAgentSuiteRunCases(suiteRunId, page, { signal }),
  }),
  control: () => ({
    queryKey: ["control"],
    queryFn: ({ signal }: Signal) => fetchAgentQualityControl({ signal }),
  }),
});

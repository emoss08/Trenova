import {
  fetchAgentQuality,
  fetchAgentQualityControl,
  fetchAgentQualityOverview,
  fetchAgentSuiteRun,
} from "@/lib/graphql/agent-quality";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type Signal = { signal?: AbortSignal };

export const agentQuality = createQueryKeys("agentQuality", {
  overview: () => ({
    queryKey: ["overview"],
    queryFn: ({ signal }: Signal) => fetchAgentQualityOverview({ signal }),
  }),
  agent: (agentDefinitionId: string) => ({
    queryKey: [agentDefinitionId],
    queryFn: ({ signal }: Signal) => fetchAgentQuality(agentDefinitionId, { signal }),
  }),
  suiteRun: (suiteRunId: string) => ({
    queryKey: [suiteRunId],
    queryFn: ({ signal }: Signal) => fetchAgentSuiteRun(suiteRunId, { signal }),
  }),
  control: () => ({
    queryKey: ["control"],
    queryFn: ({ signal }: Signal) => fetchAgentQualityControl({ signal }),
  }),
});

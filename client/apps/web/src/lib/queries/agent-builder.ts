import {
  fetchAgentDraftingAvailable,
  fetchAgentShadowReport,
  fetchAgentVersions,
  fetchInstructionLint,
  type InstructionLintRequest,
} from "@/lib/graphql/agent-builder";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const agentBuilder = createQueryKeys("agentBuilder", {
  lint: (request: InstructionLintRequest) => ({
    queryKey: [request],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchInstructionLint(request, { signal }),
  }),
  shadow: (agentId: string, days: number) => ({
    queryKey: [agentId, days],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAgentShadowReport(agentId, days, { signal }),
  }),
  drafting: () => ({
    queryKey: ["drafting"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentDraftingAvailable({ signal }),
  }),
  versions: (agentId: string) => ({
    queryKey: [agentId],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentVersions(agentId, { signal }),
  }),
});

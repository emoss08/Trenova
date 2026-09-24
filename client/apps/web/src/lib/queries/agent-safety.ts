import { fetchAgentSafetyHeaders, fetchAgentSafetySummary } from "@/lib/graphql/agent-safety";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const agentSafety = createQueryKeys("agentSafety", {
  summary: () => ({
    queryKey: ["summary"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentSafetySummary({ signal }),
  }),
  // The server answers by agent name whatever order the ids arrive in, so the
  // same agents share one entry however they were added.
  headers: (agentIds: readonly string[]) => ({
    queryKey: [[...agentIds].sort().join(",")],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAgentSafetyHeaders(agentIds, { signal }),
  }),
});

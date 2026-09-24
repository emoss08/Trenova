import { fetchAgentSafety, fetchAgentToolPolicies } from "@/lib/graphql/agent-safety";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const agentSafety = createQueryKeys("agentSafety", {
  policies: () => ({
    queryKey: ["policies"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentToolPolicies({ signal }),
  }),
  agents: () => ({
    queryKey: ["agents"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentSafety(undefined, { signal }),
  }),
});

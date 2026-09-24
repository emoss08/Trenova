import {
  fetchAgentSafety,
  fetchAgentSafetySummary,
  fetchToolPolicyPage,
  toolPolicyConnectionInput,
  type ToolPolicyFilter,
  type ToolPolicyPageRequest,
} from "@/lib/graphql/agent-safety";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const agentSafety = createQueryKeys("agentSafety", {
  summary: () => ({
    queryKey: ["summary"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentSafetySummary({ signal }),
  }),
  // Keyed by the input the server is sent, so a page is cached once however
  // the filter that produced it was spelled.
  policyPage: (filter: ToolPolicyFilter, page: ToolPolicyPageRequest) => ({
    queryKey: [toolPolicyConnectionInput(filter, page), page.includeTotalCount],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchToolPolicyPage(filter, page, { signal }),
  }),
  // One agent at a time, so adding an agent to the comparison reads only it.
  agent: (agentId: string) => ({
    queryKey: [agentId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      (await fetchAgentSafety([agentId], { signal }))[0] ?? null,
  }),
});

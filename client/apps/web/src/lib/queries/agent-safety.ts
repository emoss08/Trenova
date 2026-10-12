import {
  fetchAgentSafetyHeaders,
  fetchAgentSafetySummary,
  fetchAgentToolHolders,
  fetchToolRuleImpact,
  fetchToolRules,
  type AgentToolRuleInput,
} from "@/lib/graphql/agent-safety";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const agentSafety = createQueryKeys("agentSafety", {
  toolRules: () => ({
    queryKey: ["tool-rules"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchToolRules({ signal }),
  }),
  holders: () => ({
    queryKey: ["holders"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentToolHolders({ signal }),
  }),
  ruleImpact: (name: string, input: AgentToolRuleInput) => ({
    queryKey: [name, input],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchToolRuleImpact(name, input, { signal }),
  }),
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

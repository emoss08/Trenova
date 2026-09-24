import {
  fetchAgentChoicesByIds,
  fetchAgentDefinitions,
  fetchMyAgents,
  type AgentChoiceQuery,
  type AgentChoiceSource,
} from "@/lib/graphql/agent-definition";
import { fetchRoleAgents, fetchSuggestedAgentAudience } from "@/lib/graphql/agent-access";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const assistant = createQueryKeys("assistant", {
  threads: () => ({
    queryKey: ["assistant-threads"],
    queryFn: () => apiService.assistantService.listThreads(),
  }),
  thread: (id: string) => ({
    queryKey: ["assistant-thread", id],
    queryFn: () => apiService.assistantService.getThread(id),
  }),
  // Paged newest-first: the page param is the sequence to read above, and
  // nothing for the newest page. useThreadHistory drives it as an infinite
  // query; the key is here so the turn hook can append to the same cache.
  messages: (threadId: string) => ({
    queryKey: ["assistant-messages", threadId],
    queryFn: ({ pageParam, signal }: { pageParam?: unknown; signal?: AbortSignal }) =>
      apiService.assistantService.listMessages(threadId, {
        before: typeof pageParam === "number" ? pageParam : undefined,
        signal,
      }),
  }),
  // Every reply the person's conversations are still writing. Kept fresh by
  // the "assistant_turns" realtime event and by the turn hooks, not polled.
  activeTurns: () => ({
    queryKey: ["assistant-active-turns"],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.assistantService.listActiveTurns({ signal }),
  }),
  providers: () => ({
    queryKey: ["assistant-providers"],
    queryFn: () => apiService.assistantService.listProviders(),
  }),
  proposals: (threadId: string) => ({
    queryKey: ["assistant-proposals", threadId],
    queryFn: () => apiService.assistantService.listProposals(threadId),
  }),
  plans: (threadId: string) => ({
    queryKey: ["assistant-plans", threadId],
    queryFn: () => apiService.assistantService.listPlans(threadId),
  }),
  artifacts: (threadId: string) => ({
    queryKey: ["assistant-artifacts", threadId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.assistantService.listArtifacts(threadId, { signal }),
  }),
  // The organization's agents with everything an administrator configures;
  // AI Control only. Chat surfaces read myAgents.
  agents: (enabledOnly: boolean, chatOnly = false) => ({
    queryKey: ["agent-definitions", enabledOnly, chatOnly],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAgentDefinitions({ enabledOnly, chatOnly }, { signal }),
  }),
  // The agents the person may ask, up to a hundred, for naming the agent
  // behind each conversation.
  myAgents: () => ({
    queryKey: ["my-agents"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchMyAgents({ signal }),
  }),
  // Paged: read by useAgentChoices as an infinite query, one cursor at a time.
  agentChoices: (query: AgentChoiceQuery, source: AgentChoiceSource = "mine") => ({
    queryKey: [
      "agent-choices",
      source,
      query.search?.trim() ?? "",
      query.origin ?? "all",
      [...(query.excludeIds ?? [])],
    ],
  }),
  agentChoicesByIds: (ids: readonly string[]) => ({
    queryKey: ["agent-choices-by-id", [...ids]],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentChoicesByIds(ids, { signal }),
  }),
  // Each role's coverage of an agent's tools, for choosing who may use it.
  agentAudience: (agentId: string) => ({
    queryKey: ["agent-audience", agentId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchSuggestedAgentAudience(agentId, { signal }),
  }),
  // The agents a role is granted, for the role editor.
  roleAgents: (roleId: string) => ({
    queryKey: ["role-agents", roleId],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchRoleAgents(roleId, { signal }),
  }),
  systemAgent: (systemKey: string) => ({
    queryKey: ["agent-definition-system", systemKey],
    queryFn: () => apiService.agentDefinitionService.getBySystemKey(systemKey),
  }),
  toolCatalog: () => ({
    queryKey: ["agent-tool-catalog"],
    queryFn: () => apiService.agentDefinitionService.tools(),
  }),
  eventKinds: () => ({
    queryKey: ["agent-event-kinds"],
    queryFn: () => apiService.agentDefinitionService.eventKinds(),
  }),
  agent: (id: string) => ({
    queryKey: ["agent-definition", id],
    queryFn: () => apiService.agentDefinitionService.get(id),
  }),
  agentBudget: (id: string) => ({
    queryKey: ["agent-budget", id],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentDefinitionService.budget(id, { signal }),
  }),
  agentTrust: (id: string) => ({
    queryKey: ["agent-trust", id],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentDefinitionService.trust(id, { signal }),
  }),
  agentTemplates: () => ({
    queryKey: ["agent-templates"],
    queryFn: () => apiService.agentDefinitionService.templates(),
  }),
});

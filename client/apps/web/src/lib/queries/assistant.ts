import { fetchAgentDefinitions } from "@/lib/graphql/agent-definition";
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
  agents: (enabledOnly: boolean, chatOnly = false) => ({
    queryKey: ["agent-definitions", enabledOnly, chatOnly],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAgentDefinitions({ enabledOnly, chatOnly }, { signal }),
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

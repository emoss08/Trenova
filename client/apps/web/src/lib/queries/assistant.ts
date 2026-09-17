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
  messages: (threadId: string) => ({
    queryKey: ["assistant-messages", threadId],
    queryFn: () => apiService.assistantService.listMessages(threadId),
  }),
  proposals: (threadId: string) => ({
    queryKey: ["assistant-proposals", threadId],
    queryFn: () => apiService.assistantService.listProposals(threadId),
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
  agentTemplates: () => ({
    queryKey: ["agent-templates"],
    queryFn: () => apiService.agentDefinitionService.templates(),
  }),
});

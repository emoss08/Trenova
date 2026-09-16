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
  agents: (enabledOnly: boolean) => ({
    queryKey: ["agent-definitions", enabledOnly],
    queryFn: () => apiService.agentDefinitionService.list(enabledOnly),
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

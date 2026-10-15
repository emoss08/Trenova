import { appendToHistory, type ThreadHistory } from "@/components/assistant/thread-history";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";

/**
 * Hands a conversation to another agent: the server starts a conversation
 * with it, opened with a summary, the pinned facts and the pinned artifacts,
 * and leaves a card here saying so, which is added to this conversation's
 * history at once.
 */
export function useHandOffThread(threadId: string, facts: readonly string[] = []) {
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: (agentId: string) =>
      apiService.assistantService.handoffThread(threadId, agentId, facts),
    onSuccess: async (result) => {
      const card = result.message;
      if (card) {
        queryClient.setQueryData<ThreadHistory>(assistant.messages(threadId).queryKey, (history) =>
          appendToHistory(history, [card]),
        );
      }
      await queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey });
    },
    resourceName: "Conversation",
  });
}

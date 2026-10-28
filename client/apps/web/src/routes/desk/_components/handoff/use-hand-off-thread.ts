import { appendToHistory, type ThreadHistory } from "@/components/assistant/thread-history";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";

/** Who takes the conversation over, and, for one chapter of it, which. */
export type HandOffTarget = { agentId: string; chapterId?: string };

/**
 * Hands a conversation, or one chapter of it, to an agent: the server starts a
 * conversation with it, opened with a summary, the pinned facts and the pinned
 * artifacts, and leaves a card here saying so, which is added to this
 * conversation's history at once. A chapter may go to the conversation's own
 * agent, which carries it on in a conversation of its own.
 */
export function useHandOffThread(threadId: string, facts: readonly string[] = []) {
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: ({ agentId, chapterId }: HandOffTarget) =>
      apiService.assistantService.handoffThread(threadId, agentId, facts, chapterId),
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

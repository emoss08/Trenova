import { assistant } from "@/lib/queries/assistant";
import type { ConversationSchedule } from "@/types/assistant";
import { useQuery, type QueryClient } from "@tanstack/react-query";

/**
 * The requests scheduled in a conversation, for the question asked before it
 * is deleted: they go with it. Read by the question itself, so a list of
 * conversations reads nothing until one is about to go; null until they are
 * read, or when they could not be.
 */
export function useThreadSchedules(threadId: string): readonly ConversationSchedule[] | null {
  const query = useQuery(assistant.schedules(threadId));

  return query.data ? query.data.items : null;
}

/**
 * Drops a deleted conversation's schedules from what the browser holds. The
 * server deletes them with the conversation and announces each one, but an
 * announcement is best effort, so the Scheduled page is read again here too.
 */
export function forgetThreadSchedules(queryClient: QueryClient, threadId: string) {
  queryClient.removeQueries({ queryKey: assistant.schedules(threadId).queryKey, exact: true });
  void queryClient.invalidateQueries({ queryKey: assistant.allSchedules().queryKey });
}

import { assistant } from "@/lib/queries/assistant";
import type { AssistantThread } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";

/**
 * Deleting a conversation, asking first only when something else goes with
 * it: the requests scheduled in it. A conversation with none is deleted at
 * once. One whose schedules could not be counted is asked about, since a
 * schedule lost unasked cannot be brought back.
 *
 * `asking` is the conversation the question is open for, for
 * `DeleteConversationCard`; `confirm` and `cancel` answer it.
 */
export function useDeleteConversation(onDelete: (thread: AssistantThread) => void) {
  const queryClient = useQueryClient();
  const [asking, setAsking] = useState<AssistantThread | null>(null);
  // A second press while the first is still counting must not delete twice.
  const counting = useRef(new Set<string>());

  const ask = async (thread: AssistantThread) => {
    if (counting.current.has(thread.id)) {
      return;
    }
    counting.current.add(thread.id);
    let scheduled: number | null;
    try {
      scheduled = (await queryClient.fetchQuery(assistant.schedules(thread.id))).total;
    } catch {
      scheduled = null;
    } finally {
      counting.current.delete(thread.id);
    }

    if (scheduled === 0) {
      onDelete(thread);
    } else {
      setAsking(thread);
    }
  };

  return {
    ask: (thread: AssistantThread) => void ask(thread),
    asking,
    confirm: (thread: AssistantThread) => {
      setAsking(null);
      onDelete(thread);
    },
    cancel: () => setAsking(null),
  };
}

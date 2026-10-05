import type { AssistantThread } from "@/types/assistant";

/**
 * What a conversation's dot in the rail says, strongest first: an agent is
 * working in it, a change waits for the person's approval, its last reply
 * failed, or a reply arrived that the person has not seen. Null is a quiet
 * conversation.
 */
export type DeskThreadState = "work" | "wait" | "error" | "new" | null;

export function deskThreadState(
  thread: Pick<AssistantThread, "attention">,
  { live, active }: { live: boolean; active: boolean },
): DeskThreadState {
  if (live) {
    return "work";
  }
  const attention = thread.attention;
  if (!attention) {
    return null;
  }
  if (attention.pendingDecisions > 0) {
    return "wait";
  }
  if (attention.lastTurnFailed) {
    return "error";
  }
  if (attention.unread && !active) {
    return "new";
  }

  return null;
}

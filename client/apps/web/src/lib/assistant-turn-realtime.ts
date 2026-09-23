import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";

export const ASSISTANT_TURNS_RESOURCE = "assistant_turns";

/**
 * Whether a turn event is about somebody else's reply.
 *
 * The data channel is the whole organization's, so every member's replies
 * starting and closing arrive on every socket. Only the person's own move
 * their markers, and the event always says whose reply it is; one that does
 * not name the person, or arrives before they are known, is not theirs.
 */
export function isOtherUsersTurnEvent(
  event: ResourceInvalidationEvent,
  currentUserId: string | undefined,
): boolean {
  if (event.resource !== ASSISTANT_TURNS_RESOURCE) {
    return false;
  }
  const userId = event.entity?.userId;

  return !currentUserId || typeof userId !== "string" || userId !== currentUserId;
}

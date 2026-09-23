import type { AssistantLiveTurnList } from "@/types/assistant";

const NO_THREADS: ReadonlySet<string> = new Set();

/** The conversations with a reply still being written. */
export function liveThreadIds(list: AssistantLiveTurnList | undefined): ReadonlySet<string> {
  const items = list?.items ?? [];
  if (items.length === 0) {
    return NO_THREADS;
  }

  return new Set(items.map((turn) => turn.threadId));
}

/**
 * How many replies are being written, counted by conversation.
 *
 * A conversation writes one reply at a time, so two live turns on one thread
 * are a closing turn and its successor seen together for a moment — one reply
 * as far as the person is concerned.
 */
export function liveReplyCount(list: AssistantLiveTurnList | undefined): number {
  return liveThreadIds(list).size;
}

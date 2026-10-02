import type { AssistantLiveTurnList } from "@/types/assistant";
import type { QueryState } from "@tanstack/react-query";

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

/**
 * Whether the cached live-turn list can be taken at its word that a
 * conversation is producing no reply. Only a list the realtime connection is
 * keeping current, that has loaded, and that no event has since marked stale
 * says so; anything less is a question for the server.
 */
export function listSaysQuiet(
  state: QueryState<AssistantLiveTurnList> | undefined,
  threadId: string,
  connected: boolean,
): boolean {
  if (
    !connected ||
    state === undefined ||
    state.status !== "success" ||
    state.isInvalidated ||
    state.fetchStatus !== "idle"
  ) {
    return false;
  }

  return !liveThreadIds(state.data).has(threadId);
}

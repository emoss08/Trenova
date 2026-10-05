import {
  groupDeskThreadsByRecency,
  matchesThreadSearch,
  type DeskShelfKey,
} from "@/components/desk-chat/rail/desk-threads";
import type { AssistantThread } from "@/types/assistant";

/** A shelf of the assistant's conversation list: the ones waiting on the person, then the Desk's calendar shelves. */
export type AssistantShelfKey = "waiting" | DeskShelfKey;

export type AssistantShelf = {
  key: AssistantShelfKey;
  threads: AssistantThread[];
};

export type AssistantShelvesOptions = {
  now: number;
  timezone: string;
  /** What the person typed in the search; empty lists everything. */
  query: string;
  /** The agent a conversation is with, so a search can find it by agent. */
  agentName: (thread: AssistantThread) => string;
};

function touchedAt(thread: AssistantThread): number {
  return thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
}

function waitsOnPerson(thread: AssistantThread): boolean {
  return (thread.attention?.pendingDecisions ?? 0) > 0;
}

/**
 * The assistant's conversations as its history and sidebar list them: those
 * a change is waiting on first, then the rest shelved by when they were last
 * touched, the same shelves as the Desk's rail. Pins are the Desk's and do not
 * shelve a conversation apart here. A search narrows by title or agent.
 */
export function assistantShelves(
  threads: readonly AssistantThread[],
  { now, timezone, query, agentName }: AssistantShelvesOptions,
): AssistantShelf[] {
  const found = threads.filter((thread) => matchesThreadSearch(thread, agentName(thread), query));
  const waiting = found.filter(waitsOnPerson).sort((a, b) => touchedAt(b) - touchedAt(a));
  const rest = groupDeskThreadsByRecency(
    found.filter((thread) => !waitsOnPerson(thread)),
    now,
    { pinnedFirst: false, timezone },
  );

  return waiting.length > 0 ? [{ key: "waiting", threads: waiting }, ...rest] : rest;
}

import type { AssistantLiveTurn } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/**
 * What the beacon says, one thing at a time: a change waiting on the person,
 * a reply being written, a reply that arrived while the panel was closed, or,
 * at rest, the agent it would ask.
 */
export type BeaconMode = "idle" | "writing" | "pending" | "replied";

export type BeaconState = {
  mode: BeaconMode;
  /** Changes waiting on the person's approval. */
  pendingCount: number;
  /** Conversations with a reply being written. */
  writingCount: number;
  /** The one reply being written, when there is exactly one: its conversation and start. */
  writing: { title: string; startedAt: number } | null;
  /** Who replied (replied), or who a question would go to (idle). */
  agentName: string;
};

export type BeaconInput = {
  pendingCount: number;
  liveTurns: readonly Pick<AssistantLiveTurn, "threadId" | "threadTitle" | "startedAt">[];
  /** The agent whose reply finished while the panel was closed, if one did. */
  repliedAgentName: string | null;
  lastAgentName: string;
};

/**
 * Pending leads because only it needs the person; a reply being written comes
 * next; a finished reply holds until the panel is opened. Replies are counted
 * by conversation: a conversation writes one reply at a time, so two live
 * turns on one thread are a closing turn and its successor seen together.
 */
export function beaconState({
  pendingCount,
  liveTurns,
  repliedAgentName,
  lastAgentName,
}: BeaconInput): BeaconState {
  const latestByThread = new Map<string, { title: string; startedAt: number }>();
  for (const turn of liveTurns) {
    const seen = latestByThread.get(turn.threadId);
    if (!seen || turn.startedAt > seen.startedAt) {
      latestByThread.set(turn.threadId, { title: turn.threadTitle, startedAt: turn.startedAt });
    }
  }
  const pending = Math.max(0, pendingCount);
  const writingCount = latestByThread.size;
  const writing = writingCount === 1 ? ([...latestByThread.values()][0] ?? null) : null;

  if (pending > 0) {
    return {
      mode: "pending",
      pendingCount: pending,
      writingCount,
      writing,
      agentName: lastAgentName,
    };
  }
  if (writingCount > 0) {
    return { mode: "writing", pendingCount: 0, writingCount, writing, agentName: lastAgentName };
  }
  if (repliedAgentName !== null) {
    return {
      mode: "replied",
      pendingCount: 0,
      writingCount: 0,
      writing: null,
      agentName: repliedAgentName,
    };
  }

  return {
    mode: "idle",
    pendingCount: 0,
    writingCount: 0,
    writing: null,
    agentName: lastAgentName,
  };
}

/** The beacon's accessible name: everything it shows, with the counts uncapped. */
export function launcherLabel(t: TranslateFn, state: BeaconState): string {
  const { pendingCount, writingCount } = state;
  if (pendingCount > 0 && writingCount > 0) {
    return t(
      "Open the assistant, {0} changes await your decision, {1, plural, one {# reply is} other {# replies are}} being written",
      pendingCount,
      writingCount,
    );
  }
  if (pendingCount > 0) {
    return t("Open the assistant, {0} changes await your decision", pendingCount);
  }
  if (writingCount > 0) {
    return t(
      "Open the assistant, {0, plural, one {# reply is} other {# replies are}} being written",
      writingCount,
    );
  }
  if (state.mode === "replied") {
    return t("Open the assistant, {0} replied", state.agentName);
  }

  return t("Open the assistant");
}

/** A count that would stretch the beacon, capped for the eye. */
export function cappedCount(count: number): string {
  return count > 99 ? "99+" : String(count);
}

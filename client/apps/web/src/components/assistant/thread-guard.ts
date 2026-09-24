import type { AgentChoice } from "@/lib/graphql/agent-definition";

/**
 * Why the composer is closed, or null when it is open. `read-only` is the
 * one that replaces the composer rather than disabling it: the server says
 * the reader may no longer ask this conversation's agent anything, so the
 * conversation is a record to read, not a box waiting to be typed in.
 */
export type ComposerBlock = "read-only" | "full" | "agent-missing" | "agents-unavailable";

/**
 * Decides whether a person can type, and if not, why.
 *
 * The server's word comes first: a conversation it serves with
 * `canContinue` false is read-only whatever the list of agents says, because
 * that list is read at another moment and the server is the one that would
 * refuse the message.
 *
 * Otherwise the agent behind a thread is looked up in the list of chat
 * agents the person may ask. When that list could not be fetched at all,
 * the agent is not missing, it is unknown, and telling someone their agent
 * was switched off sends them to AI Control to fix a thing that is not
 * broken there.
 */
export function composerBlock({
  agent,
  agentsUnavailable,
  threadFull,
  canContinue = true,
}: {
  agent: AgentChoice | null;
  agentsUnavailable: boolean;
  threadFull: boolean;
  /** The thread's own `canContinue`, as the server served it. */
  canContinue?: boolean;
}): ComposerBlock | null {
  if (!canContinue) {
    return "read-only";
  }
  if (threadFull) {
    return "full";
  }
  if (agent !== null) {
    return null;
  }

  return agentsUnavailable ? "agents-unavailable" : "agent-missing";
}

/**
 * Whether a question asked before its thread existed should be sent now.
 *
 * The Desk's front page opens a conversation and hands it the question that
 * prompted it, which means the send happens on the other side of a
 * navigation from the ask. Three things have to hold, and each of them is a
 * way this went wrong before it was written down:
 *
 *   - there is a question, and it is not blank;
 *   - it has not already been sent from this mount, or a re-render that
 *     changes any dependency sends it again;
 *   - the thread's history has loaded and is empty, or a reload with the
 *     question still in hand asks it a second time into a thread that has
 *     already answered it.
 */
export function shouldSendOpeningQuestion({
  question,
  alreadySent,
  historyLoading,
  messageCount,
}: {
  question: string | undefined;
  alreadySent: boolean;
  historyLoading: boolean;
  messageCount: number;
}): boolean {
  if (question === undefined || question.trim() === "" || alreadySent) {
    return false;
  }

  return !historyLoading && messageCount === 0;
}

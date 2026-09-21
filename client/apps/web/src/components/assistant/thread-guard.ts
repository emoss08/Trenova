import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";

/** Why the composer is closed, or null when it is open. */
export type ComposerBlock = "full" | "agent-missing" | "agents-unavailable";

/**
 * Decides whether a person can type, and if not, why.
 *
 * The agent behind a thread is looked up in the list of enabled chat
 * agents. When that list could not be fetched at all, the agent is not
 * disabled, it is unknown, and telling someone their agent was switched off
 * sends them to AI Control to fix a thing that is not broken there.
 */
export function composerBlock({
  agent,
  agentsUnavailable,
  threadFull,
}: {
  agent: AgentDefinitionRow | null;
  agentsUnavailable: boolean;
  threadFull: boolean;
}): ComposerBlock | null {
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

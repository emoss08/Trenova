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

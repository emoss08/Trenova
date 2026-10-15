import {
  AGENT_MATCH_MAX_CONTEXT_KINDS,
  AGENT_RECOMMENDATION_MAX_LENGTH,
  type AgentChoice,
  type AgentMatch,
} from "@/lib/graphql/agent-definition";
import type { DeskSettings } from "@/stores/desk-settings-store";
import type { AssistantEntityRef, AssistantPageContext } from "@/types/assistant";

/** How long typing has to pause before the question is matched to an agent. */
export const RECOMMENDATION_SETTLE_MS = 450;

/**
 * The longest a question sent before its match came back waits for it. Past
 * this it goes to the agent already chosen, so a slow or failing match never
 * holds a question up.
 */
export const RECOMMENDATION_WAIT_MS = 1500;

/** How long a match for the same words and agent is reused rather than asked again. */
export const RECOMMENDATION_STALE_MS = 60_000;

/** The fewest characters worth matching: fewer name nothing a tool could be found by. */
export const RECOMMENDATION_MIN_LENGTH = 8;

/**
 * The words a question is matched to an agent on, or null when there is
 * nothing worth matching yet: too short to name a task, or a slash command
 * being typed, which the composer handles itself.
 */
export function recommendationQueryText(draft: string): string | null {
  const text = draft.trim();
  if (text.length < RECOMMENDATION_MIN_LENGTH || text.startsWith("/")) {
    return null;
  }

  return text.length > AGENT_RECOMMENDATION_MAX_LENGTH
    ? text.slice(0, AGENT_RECOMMENDATION_MAX_LENGTH)
    : text;
}

/** A match that names another agent holding the tools the question needs. */
export type OfferedMatch = AgentMatch & { outcome: "BetterAgent"; agent: AgentChoice };

/**
 * What the person's settings say to do when another agent fits a question
 * better: move it there without asking (`auto`), name it and send to it
 * (`switch`), only name it (`suggest`), or nothing (`off`).
 */
export type AgentSuggestionMode = "auto" | "switch" | "suggest" | "off";

export function suggestionMode(
  settings: Pick<DeskSettings, "autoSwitchAgent" | "suggestAgent">,
): AgentSuggestionMode {
  return settings.autoSwitchAgent === "on" ? "auto" : settings.suggestAgent;
}

/** Whether the note naming a better agent is drawn above the box. */
export function showsSuggestion(mode: AgentSuggestionMode): boolean {
  return mode === "switch" || mode === "suggest";
}

/** Whether a question sent goes to the better agent, unless the person picked one. */
export function routesOnSend(mode: AgentSuggestionMode): boolean {
  return mode === "auto" || mode === "switch";
}

/**
 * Whether a match is worth offering beside the composer: it names an agent
 * other than the one the question would go to now, and not one the person
 * already turned down with the cross while writing this question.
 */
export function offersAgent(
  match: AgentMatch | null,
  agentId: string | null,
  keptOver: readonly string[] = [],
): match is OfferedMatch {
  return (
    match !== null &&
    match.outcome === "BetterAgent" &&
    match.agent !== null &&
    match.agent.id !== agentId &&
    !keptOver.includes(match.agent.id)
  );
}

/** Whether tools match the question but none of the person's agents holds them. */
export function noAgentFits(match: AgentMatch | null): boolean {
  return match?.outcome === "NoAgent";
}

/**
 * The agent a question goes to when it is sent. An agent the person picked
 * is where it goes, and so is the agent chosen when their settings only name
 * a better one. Otherwise it goes to the agent that holds the tools it needs,
 * when there is one they have not turned down, and to the agent already
 * chosen when there is none.
 */
export function agentToAsk({
  agent,
  match,
  pinned,
  keptOver,
  mode,
}: {
  agent: AgentChoice;
  match: AgentMatch | null;
  /** The person picked the agent themselves. */
  pinned: boolean;
  keptOver: readonly string[];
  mode: AgentSuggestionMode;
}): AgentChoice {
  if (pinned || !routesOnSend(mode) || !offersAgent(match, agent.id, keptOver)) {
    return agent;
  }

  return match.agent;
}

/**
 * The kinds of record a question is about, for ranking the tools that work on
 * them first: the records it mentions, then the record or list the page it is
 * asked from shows.
 */
export function matchContextKinds(
  mentions: readonly Pick<AssistantEntityRef, "type">[],
  page: Pick<AssistantPageContext, "entityType" | "view"> | null,
): string[] {
  const kinds: string[] = [];
  const add = (kind: string | undefined | null) => {
    if (kind && !kinds.includes(kind) && kinds.length < AGENT_MATCH_MAX_CONTEXT_KINDS) {
      kinds.push(kind);
    }
  };
  for (const mention of mentions) {
    add(mention.type);
  }
  add(page?.entityType);
  add(page?.view?.resource);

  return kinds;
}

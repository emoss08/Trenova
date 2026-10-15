import {
  AGENT_RECOMMENDATION_MAX_LENGTH,
  type AgentChoice,
  type AgentRecommendation,
} from "@/lib/graphql/agent-definition";

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

/**
 * Whether a match is worth offering beside the composer: it names an agent
 * other than the one the question would go to now, and the person has not
 * already said to keep theirs over that agent.
 */
export function offersRecommendation(
  recommendation: AgentRecommendation | null,
  agentId: string | null,
  keptOverId: string | null,
): recommendation is AgentRecommendation {
  return (
    recommendation !== null &&
    !recommendation.current &&
    recommendation.agent.id !== agentId &&
    recommendation.agent.id !== keptOverId
  );
}

/**
 * The agent a question goes to when it is sent. Once the person has chosen,
 * from the picker or from the suggestion, it goes where they chose. Until
 * then it goes to the agent that holds the tools it needs, when there is one,
 * and otherwise to the agent already chosen.
 */
export function agentToAsk({
  agent,
  recommendation,
  chosen,
}: {
  agent: AgentChoice;
  recommendation: AgentRecommendation | null;
  chosen: boolean;
}): AgentChoice {
  if (chosen || !offersRecommendation(recommendation, agent.id, null)) {
    return agent;
  }

  return recommendation.agent;
}

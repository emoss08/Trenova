import type { TurnHandoff } from "@/components/assistant/turn-stream";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantHandoff, AssistantMessage } from "@/types/assistant";

/** No agent is suggested. One shared value, so an unchanged answer never re-renders the menu. */
export const NO_HANDOFF_SUGGESTION: readonly string[] = Object.freeze([]);

type SuggestionMessage = Pick<AssistantMessage, "role" | "kind" | "handOffAgents">;

/**
 * The agents the conversation's agent last said hold what it could not do:
 * the newest find_tools result naming any, so long as the person has not
 * written since. A newer message from the person is a new question, which
 * the old suggestion may not answer. A step another agent took on a task it
 * was handed is not the conversation's agent speaking, so it is passed over.
 *
 * Pages are the history's own: newest first, each in reading order. The walk
 * stops at the first answer, so an ordinary thread reads only its last few
 * messages.
 *
 * The turn being written, when there is one, is newer than anything saved, so
 * it is read first: its own find_tools answer naming agents is the
 * suggestion before the turn is saved, and a turn answering words of the
 * person's own is them writing again, so nothing older stands. A turn
 * following up a decision carries no words of theirs; until it names an
 * agent the saved history decides.
 */
export function handoffSuggestion(
  pages: readonly { results: readonly SuggestionMessage[] }[],
  live: TurnHandoff | null = null,
): readonly string[] {
  if (live !== null) {
    if (live.agentIds.length > 0) {
      return live.agentIds;
    }
    if (live.asked) {
      return NO_HANDOFF_SUGGESTION;
    }
  }
  for (const page of pages) {
    for (let index = page.results.length - 1; index >= 0; index -= 1) {
      const message = page.results[index];
      if (message.role === "User" && (message.kind === "Message" || message.kind === "Steer")) {
        return NO_HANDOFF_SUGGESTION;
      }
      if (
        message.role === "Tool" &&
        message.kind !== "Delegated" &&
        message.handOffAgents &&
        message.handOffAgents.length > 0
      ) {
        return message.handOffAgents;
      }
    }
  }

  return NO_HANDOFF_SUGGESTION;
}

/**
 * The agents a conversation can be handed to: every agent the person may use
 * except the one it is with. The agents the conversation's agent named as
 * holding what it could not do come first, in the order it named them, since
 * they are the ones that can answer; then the agents it already hands work
 * to, in their configured order, since they are the ones it would send the
 * question to itself; then everyone else as they came.
 */
export function handoffTargets(
  agents: readonly AgentChoice[],
  current: AgentChoice | null,
  suggested: readonly string[] = NO_HANDOFF_SUGGESTION,
): AgentChoice[] {
  const others = agents.filter((agent) => agent.id !== current?.id);
  const suggestedOrder = new Map(suggested.map((id, index) => [id, index]));
  const delegateOrder = new Map((current?.delegates ?? []).map((mark, index) => [mark.id, index]));
  const rank = (id: string): [number, number] => {
    const suggestion = suggestedOrder.get(id);
    if (suggestion !== undefined) return [0, suggestion];
    const delegate = delegateOrder.get(id);
    if (delegate !== undefined) return [1, delegate];
    return [2, 0];
  };

  return others
    .map((agent, index) => ({ agent, index, rank: rank(agent.id) }))
    .sort((a, b) => a.rank[0] - b.rank[0] || a.rank[1] - b.rank[1] || a.index - b.index)
    .map(({ agent }) => agent);
}

export type HandoffMenuState = {
  open: boolean;
  /** The agent a hand-off is being made to; null while none is. */
  pendingAgentId: string | null;
};

export type HandoffMenuAction =
  | { type: "toggle" }
  | { type: "close" }
  | { type: "pick"; agentId: string }
  | { type: "settled" };

export const HANDOFF_MENU_CLOSED: HandoffMenuState = { open: false, pendingAgentId: null };

/**
 * The menu's state. Picking an agent closes it and holds the pick until the
 * request settles, so a second pick while one is in flight is ignored rather
 * than making two conversations.
 */
export function handoffMenuReducer(
  state: HandoffMenuState,
  action: HandoffMenuAction,
): HandoffMenuState {
  switch (action.type) {
    case "toggle":
      return state.pendingAgentId === null ? { ...state, open: !state.open } : state;
    case "close":
      return state.open ? { ...state, open: false } : state;
    case "pick":
      return state.pendingAgentId === null
        ? { open: false, pendingAgentId: action.agentId }
        : state;
    case "settled":
      return { ...state, pendingAgentId: null };
  }
}

/** What a hand-off carried, as the card's chips name it. */
export type CarriedChip =
  | { kind: "summary" }
  | { kind: "facts"; count: number }
  | { kind: "artifact"; title: string };

export function carriedChips(handoff: AssistantHandoff): CarriedChip[] {
  const chips: CarriedChip[] = [];
  if (handoff.summary.trim() !== "") {
    chips.push({ kind: "summary" });
  }
  if (handoff.facts.length > 0) {
    chips.push({ kind: "facts", count: handoff.facts.length });
  }
  for (const artifact of handoff.artifacts) {
    chips.push({ kind: "artifact", title: artifact.title });
  }

  return chips;
}

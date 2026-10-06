import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantHandoff } from "@/types/assistant";

/**
 * The agents a conversation can be handed to: every agent the person may use
 * except the one it is with. The agents the current one already hands work
 * to come first, in their configured order, since they are the ones it
 * would send the question to itself.
 */
export function handoffTargets(
  agents: readonly AgentChoice[],
  current: AgentChoice | null,
): AgentChoice[] {
  const others = agents.filter((agent) => agent.id !== current?.id);
  const delegateOrder = new Map((current?.delegates ?? []).map((mark, index) => [mark.id, index]));

  return others
    .map((agent, index) => ({ agent, index }))
    .sort((a, b) => {
      const da = delegateOrder.get(a.agent.id) ?? Number.POSITIVE_INFINITY;
      const db = delegateOrder.get(b.agent.id) ?? Number.POSITIVE_INFINITY;
      return da === db ? a.index - b.index : da - db;
    })
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

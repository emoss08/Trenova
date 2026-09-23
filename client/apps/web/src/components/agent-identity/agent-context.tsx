import { createContext, use, useMemo, type ReactNode } from "react";
import type { AgentIdentityInput } from "./agent-identity";

type AssistantAgentValue = {
  agent: AgentIdentityInput | null;
  /** The agents the thread's agent may hand work to, by id, so a hand-off draws its mark. */
  delegates: ReadonlyMap<string, AgentIdentityInput>;
};

const NO_DELEGATES: ReadonlyMap<string, AgentIdentityInput> = new Map();

const AssistantAgentContext = createContext<AssistantAgentValue>({
  agent: null,
  delegates: NO_DELEGATES,
});

/**
 * The agent a thread belongs to. Every turn in the thread draws that agent's
 * mark, but only the thread knows which agent it is, so it publishes it rather
 * than threading it through every message component. The agents it may hand
 * work to ride along, because a saved hand-off names its agent and nothing
 * else about it.
 */
export function AssistantAgentProvider({
  agent,
  delegates,
  children,
}: {
  agent: AgentIdentityInput | null;
  delegates?: readonly AgentIdentityInput[] | null;
  children: ReactNode;
}) {
  const value = useMemo<AssistantAgentValue>(() => {
    if (!delegates || delegates.length === 0) {
      return { agent, delegates: NO_DELEGATES };
    }
    const byId = new Map<string, AgentIdentityInput>();
    for (const delegate of delegates) {
      if (delegate.id) {
        byId.set(delegate.id, delegate);
      }
    }
    return { agent, delegates: byId };
  }, [agent, delegates]);

  return <AssistantAgentContext value={value}>{children}</AssistantAgentContext>;
}

export function useAssistantAgent(): AgentIdentityInput | null {
  return use(AssistantAgentContext).agent;
}

/**
 * Another agent as a hand-off draws it: its mark from the thread's list of
 * agents it may ask, or from what the hand-off itself carries, which is
 * always its id and name and, while it streams, its icon and accent. An
 * agent removed from the list since still has a mark of its own, derived
 * from its id.
 */
export function useDelegateIdentity(fallback: AgentIdentityInput): AgentIdentityInput {
  const { delegates } = use(AssistantAgentContext);
  const known = fallback.id ? delegates.get(fallback.id) : undefined;

  return useMemo(() => {
    if (!known) {
      return fallback;
    }
    return {
      id: known.id,
      name: known.name || fallback.name,
      icon: known.icon || fallback.icon,
      accent: known.accent || fallback.accent,
      template: known.template ?? fallback.template,
    };
  }, [fallback, known]);
}

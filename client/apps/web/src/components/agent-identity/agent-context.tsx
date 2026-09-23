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
 * work to ride along, for a hand-off that carries no mark of its own: one
 * saved before the thread served agents' marks.
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
 * Another agent as a hand-off draws it. What the hand-off itself carries
 * comes first: always its id and name, and its icon and accent as the stream
 * announced them or the thread served them. Where it carries no mark, the
 * thread's list of agents it may ask fills it in; and an agent in neither
 * still has a mark of its own, derived from its id.
 */
export function useDelegateIdentity(own: AgentIdentityInput): AgentIdentityInput {
  const { delegates } = use(AssistantAgentContext);
  const known = own.id ? delegates.get(own.id) : undefined;

  return useMemo(() => {
    if (!known) {
      return own;
    }
    return {
      id: known.id,
      name: own.name || known.name,
      icon: own.icon || known.icon,
      accent: own.accent || known.accent,
      template: own.template ?? known.template,
    };
  }, [own, known]);
}

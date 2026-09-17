import { createContext, use, type ReactNode } from "react";
import type { AgentIdentityInput } from "./agent-identity";

const AssistantAgentContext = createContext<AgentIdentityInput | null>(null);

/**
 * The agent a thread belongs to. Every turn in the thread draws that agent's
 * mark, but only the thread knows which agent it is, so it publishes it rather
 * than threading it through every message component.
 */
export function AssistantAgentProvider({
  agent,
  children,
}: {
  agent: AgentIdentityInput | null;
  children: ReactNode;
}) {
  return <AssistantAgentContext value={agent}>{children}</AssistantAgentContext>;
}

export function useAssistantAgent(): AgentIdentityInput | null {
  return use(AssistantAgentContext);
}

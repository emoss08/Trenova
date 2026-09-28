import { createContext, useContext, type ReactNode } from "react";

/**
 * Sends a message into the conversation on the person's behalf, as if they
 * had typed it: a card that wants the agent to fix its proposal says so
 * through this. Null outside a conversation, or in one that can no longer
 * continue; a surface without it offers the reasons another way.
 */
export type AskAgent = (message: string) => void;

const AskAgentContext = createContext<AskAgent | null>(null);

export function AskAgentProvider({
  value,
  children,
}: {
  value: AskAgent | null;
  children: ReactNode;
}) {
  return <AskAgentContext value={value}>{children}</AskAgentContext>;
}

export function useAskAgent(): AskAgent | null {
  return useContext(AskAgentContext);
}
